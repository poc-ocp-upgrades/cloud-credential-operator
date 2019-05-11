package credentialsrequest

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"
	log "github.com/sirupsen/logrus"
	configv1 "github.com/openshift/api/config/v1"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/openshift/cloud-credential-operator/pkg/controller/utils"
	"github.com/openshift/cloud-credential-operator/pkg/util/clusteroperator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	cloudCredOperatorNamespace		= "openshift-cloud-credential-operator"
	cloudCredClusterOperator		= "cloud-credential"
	reasonCredentialsFailing		= "CredentialsFailing"
	reasonNoCredentialsFailing		= "NoCredentialsFailing"
	reasonReconciling				= "Reconciling"
	reasonReconcilingComplete		= "ReconcilingComplete"
	reasonCredentialsNotProvisioned	= "CredentialsNotProvisioned"
)

func (r *ReconcileCredentialsRequest) syncOperatorStatus() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	log.Debug("syncing cluster operator status")
	co := &configv1.ClusterOperator{ObjectMeta: metav1.ObjectMeta{Name: cloudCredClusterOperator}}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: co.Name}, co)
	isNotFound := errors.IsNotFound(err)
	if err != nil && !isNotFound {
		return fmt.Errorf("failed to get clusteroperator %s: %v", co.Name, err)
	}
	_, credRequests, err := r.getOperatorState()
	if err != nil {
		return fmt.Errorf("failed to get operator state: %v", err)
	}
	oldConditions := co.Status.Conditions
	oldVersions := co.Status.Versions
	co.Status.Conditions = computeStatusConditions(oldConditions, credRequests)
	co.Status.Versions = computeClusterOperatorVersions()
	if isNotFound {
		if err := r.Client.Create(context.TODO(), co); err != nil {
			return fmt.Errorf("failed to create clusteroperator %s: %v", co.Name, err)
		}
		log.Info("created clusteroperator")
	}
	if !reflect.DeepEqual(oldVersions, co.Status.Versions) {
		log.WithFields(log.Fields{"old": oldVersions, "new": co.Status.Versions}).Info("version has changed, updating progressing condition lastTransitionTime")
		progressing := findClusterOperatorCondition(co.Status.Conditions, configv1.OperatorProgressing)
		progressing.LastTransitionTime = metav1.Time{Time: time.Now()}
	}
	if !clusteroperator.ConditionsEqual(oldConditions, co.Status.Conditions) || !reflect.DeepEqual(oldVersions, co.Status.Versions) {
		err = r.Client.Status().Update(context.TODO(), co)
		if err != nil {
			return fmt.Errorf("failed to update clusteroperator %s: %v", co.Name, err)
		}
		log.Debug("cluster operator status updated")
	}
	return nil
}
func (r *ReconcileCredentialsRequest) getOperatorState() (*corev1.Namespace, []minterv1.CredentialsRequest, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	ns := &corev1.Namespace{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cloudCredOperatorNamespace}, ns); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("error getting Namespace %s: %v", cloudCredOperatorNamespace, err)
	}
	credRequestList := &minterv1.CredentialsRequestList{}
	err := r.Client.List(context.TODO(), &client.ListOptions{Namespace: cloudCredOperatorNamespace}, credRequestList)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list CredentialsRequests: %v", err)
	}
	return ns, credRequestList.Items, nil
}
func computeClusterOperatorVersions() []configv1.OperandVersion {
	_logClusterCodePath()
	defer _logClusterCodePath()
	currentVersion := os.Getenv("RELEASE_VERSION")
	versions := []configv1.OperandVersion{{Name: "operator", Version: currentVersion}}
	return versions
}
func computeStatusConditions(conditions []configv1.ClusterOperatorStatusCondition, credRequests []minterv1.CredentialsRequest) []configv1.ClusterOperatorStatusCondition {
	_logClusterCodePath()
	defer _logClusterCodePath()
	failingCondition := &configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorFailing, Status: configv1.ConditionFalse}
	failureConditionTypes := []minterv1.CredentialsRequestConditionType{minterv1.InsufficientCloudCredentials, minterv1.MissingTargetNamespace, minterv1.CredentialsProvisionFailure, minterv1.CredentialsDeprovisionFailure}
	failingCredRequests := 0
	for _, cr := range credRequests {
		foundFailure := false
		for _, t := range failureConditionTypes {
			failureCond := utils.FindCredentialsRequestCondition(cr.Status.Conditions, t)
			if failureCond != nil && failureCond.Status == corev1.ConditionTrue {
				foundFailure = true
				break
			}
		}
		if foundFailure {
			failingCredRequests = failingCredRequests + 1
		}
	}
	if failingCredRequests > 0 {
		failingCondition.Status = configv1.ConditionTrue
		failingCondition.Reason = reasonCredentialsFailing
		failingCondition.Message = fmt.Sprintf("%d of %d credentials requests are failing to sync.", failingCredRequests, len(credRequests))
	} else {
		failingCondition.Status = configv1.ConditionFalse
		failingCondition.Reason = reasonNoCredentialsFailing
		failingCondition.Message = "No credentials requests reporting errors."
	}
	conditions = clusteroperator.SetStatusCondition(conditions, failingCondition)
	progressingCondition := &configv1.ClusterOperatorStatusCondition{Type: configv1.OperatorProgressing, Status: configv1.ConditionUnknown}
	credRequestsNotProvisioned := 0
	log.Debugf("%d cred requests", len(credRequests))
	for _, cr := range credRequests {
		if !cr.Status.Provisioned {
			credRequestsNotProvisioned = credRequestsNotProvisioned + 1
		}
	}
	if credRequestsNotProvisioned > 0 || failingCredRequests > 0 {
		progressingCondition.Status = configv1.ConditionTrue
		progressingCondition.Reason = reasonReconciling
		progressingCondition.Message = fmt.Sprintf("%d of %d credentials requests provisioned, %d reporting errors.", len(credRequests)-credRequestsNotProvisioned, len(credRequests), failingCredRequests)
	} else {
		progressingCondition.Status = configv1.ConditionFalse
		progressingCondition.Reason = reasonReconcilingComplete
		progressingCondition.Message = fmt.Sprintf("%d of %d credentials requests provisioned and reconciled.", len(credRequests), len(credRequests))
	}
	conditions = clusteroperator.SetStatusCondition(conditions, progressingCondition)
	availableCondition := &configv1.ClusterOperatorStatusCondition{Status: configv1.ConditionTrue, Type: configv1.OperatorAvailable}
	conditions = clusteroperator.SetStatusCondition(conditions, availableCondition)
	for _, c := range conditions {
		log.WithFields(log.Fields{"type": c.Type, "status": c.Status, "reason": c.Reason, "message": c.Message}).Debug("set ClusterOperator condition")
	}
	return conditions
}
func findClusterOperatorCondition(conditions []configv1.ClusterOperatorStatusCondition, conditionType configv1.ClusterStatusConditionType) *configv1.ClusterOperatorStatusCondition {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
