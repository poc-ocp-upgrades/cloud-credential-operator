package credentialsrequest

import (
	"context"
	"bytes"
	"net/http"
	"runtime"
	"fmt"
	"reflect"
	"time"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	"github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest/actuator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/internalcontroller"
	"github.com/openshift/cloud-credential-operator/pkg/controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	namespaceMissing		= "NamespaceMissing"
	namespaceExists			= "NamespaceExists"
	cloudCredsInsufficient		= "CloudCredsInsufficient"
	cloudCredsSufficient		= "CloudCredsSufficient"
	credentialsProvisionFailure	= "CredentialsProvisionFailure"
	credentialsProvisionSuccess	= "CredentialsProvisionSuccess"
	cloudCredDeprovisionFailure	= "CloudCredDeprovisionFailure"
	cloudCredDeprovisionSuccess	= "CloudCredDeprovisionSuccess"
)

func AddWithActuator(mgr manager.Manager, actuator actuator.Actuator) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return add(mgr, newReconciler(mgr, actuator))
}
func newReconciler(mgr manager.Manager, actuator actuator.Actuator) reconcile.Reconciler {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &ReconcileCredentialsRequest{Client: mgr.GetClient(), Actuator: actuator}
}
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := mgr.SetFields(r); err != nil {
		return err
	}
	name := "credentialsrequest-controller"
	rateLimiter := workqueue.NewMaxOfRateLimiter(workqueue.NewItemExponentialFailureRateLimiter(2*time.Second, 1000*time.Second), &workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)})
	c := &internalcontroller.Controller{Do: r, Cache: mgr.GetCache(), Config: mgr.GetConfig(), Scheme: mgr.GetScheme(), Client: mgr.GetClient(), Recorder: mgr.GetRecorder(name), Queue: workqueue.NewNamedRateLimitingQueue(rateLimiter, name), MaxConcurrentReconciles: 1, Name: name}
	if err := mgr.Add(c); err != nil {
		return err
	}
	err := c.Watch(&source.Kind{Type: &minterv1.CredentialsRequest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	mapFn := handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		namespace, name, err := cache.SplitMetaNamespaceKey(a.Meta.GetAnnotations()[minterv1.AnnotationCredentialsRequest])
		if err != nil {
			log.WithField("labels", a.Meta.GetAnnotations()).WithError(err).Error("error splitting namespace key for label")
			return []reconcile.Request{}
		}
		log.WithField("cr", fmt.Sprintf("%s/%s", namespace, name)).Debug("parsed annotation")
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: name, Namespace: namespace}}}
	})
	p := predicate.Funcs{UpdateFunc: func(e event.UpdateEvent) bool {
		if _, ok := e.MetaOld.GetAnnotations()[minterv1.AnnotationCredentialsRequest]; !ok {
			return false
		}
		return true
	}, CreateFunc: func(e event.CreateEvent) bool {
		if _, ok := e.Meta.GetAnnotations()[minterv1.AnnotationCredentialsRequest]; !ok {
			return false
		}
		return true
	}, DeleteFunc: func(e event.DeleteEvent) bool {
		if _, ok := e.Meta.GetAnnotations()[minterv1.AnnotationCredentialsRequest]; !ok {
			return false
		}
		return true
	}}
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}, p)
	if err != nil {
		return err
	}
	namespaceMapFn := handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		newNamespace := a.Meta.GetName()
		log.WithField("namespace", newNamespace).Debug("checking for credentials requests targeting namespace")
		crs := &minterv1.CredentialsRequestList{}
		mgr.GetClient().List(context.TODO(), &client.ListOptions{}, crs)
		requests := []reconcile.Request{}
		for _, cr := range crs.Items {
			if !cr.Status.Provisioned && cr.Spec.SecretRef.Namespace == newNamespace {
				log.WithFields(log.Fields{"namespace": newNamespace, "cr": fmt.Sprintf("%s/%s", cr.Spec.SecretRef.Namespace, cr.Spec.SecretRef.Name)}).Info("found credentials request for namespace")
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}})
			}
		}
		return requests
	})
	namespacePred := predicate.Funcs{CreateFunc: func(e event.CreateEvent) bool {
		return true
	}, UpdateFunc: func(e event.UpdateEvent) bool {
		return false
	}, DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	}}
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: namespaceMapFn}, namespacePred)
	if err != nil {
		return err
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcileCredentialsRequest{}

type ReconcileCredentialsRequest struct {
	client.Client
	Actuator	actuator.Actuator
}

func (r *ReconcileCredentialsRequest) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := log.WithFields(log.Fields{"controller": "credreq", "cr": fmt.Sprintf("%s/%s", request.NamespacedName.Namespace, request.NamespacedName.Name)})
	defer func() {
		err := r.syncOperatorStatus()
		if err != nil {
			logger.WithError(err).Error("failed to sync operator status")
		}
	}()
	logger.Info("syncing credentials request")
	cr := &minterv1.CredentialsRequest{}
	err := r.Get(context.TODO(), request.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug("credentials request no longer exists")
			return reconcile.Result{}, nil
		}
		logger.WithError(err).Error("error getting credentials request, requeuing")
		return reconcile.Result{}, err
	}
	logger = logger.WithFields(log.Fields{"secret": fmt.Sprintf("%s/%s", cr.Spec.SecretRef.Namespace, cr.Spec.SecretRef.Name)})
	origCR := cr
	cr = cr.DeepCopy()
	if cr.DeletionTimestamp != nil {
		if HasFinalizer(cr, minterv1.FinalizerDeprovision) {
			err = r.Actuator.Delete(context.TODO(), cr)
			if err != nil {
				logger.WithError(err).Error("actuator error deleting credentials exist")
				setCredentialsDeprovisionFailureCondition(cr, true, err)
				if err := r.updateStatus(origCR, cr, logger); err != nil {
					logger.WithError(err).Error("failed to update condition")
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			} else {
				setCredentialsDeprovisionFailureCondition(cr, false, nil)
				if err := r.updateStatus(origCR, cr, logger); err != nil {
					logger.Warnf("unable to update condition: %v", err)
				}
			}
			targetSecret := &corev1.Secret{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: cr.Spec.SecretRef.Namespace, Name: cr.Spec.SecretRef.Name}, targetSecret)
			sLog := logger.WithFields(log.Fields{"targetSecret": fmt.Sprintf("%s/%s", cr.Spec.SecretRef.Namespace, cr.Spec.SecretRef.Name)})
			if err != nil {
				if errors.IsNotFound(err) {
					sLog.Debug("target secret does not exist")
				} else {
					sLog.WithError(err).Error("unexpected error getting target secret to delete")
					return reconcile.Result{}, err
				}
			} else {
				err := r.Client.Delete(context.TODO(), targetSecret)
				if err != nil {
					sLog.WithError(err).Error("error deleting target secret")
					return reconcile.Result{}, err
				} else {
					sLog.Info("target secret deleted successfully")
				}
			}
			logger.Info("actuator deletion complete, removing finalizer")
			err = r.removeDeprovisionFinalizer(cr)
			if err != nil {
				logger.WithError(err).Error("error removing deprovision finalizer")
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		} else {
			logger.Info("credentials request deleted and finalizer no longer present, nothing to do")
			return reconcile.Result{}, nil
		}
	} else {
		if !HasFinalizer(cr, minterv1.FinalizerDeprovision) {
			logger.Infof("adding finalizer: %s", minterv1.FinalizerDeprovision)
			err = r.addDeprovisionFinalizer(cr)
			if err != nil {
				logger.WithError(err).Error("error adding finalizer")
			}
			return reconcile.Result{}, err
		}
	}
	targetNS := &corev1.Namespace{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: cr.Spec.SecretRef.Namespace}, targetNS)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Warn("secret namespace does not yet exist")
			setMissingTargetNamespaceCondition(cr, true)
			if err := r.updateStatus(origCR, cr, logger); err != nil {
				logger.WithError(err).Error("error updating condition")
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		logger.WithError(err).Error("unexpected error looking up namespace")
		return reconcile.Result{}, err
	} else {
		logger.Debug("found secret namespace")
		setMissingTargetNamespaceCondition(cr, false)
	}
	isStale := cr.Generation != cr.Status.LastSyncGeneration
	hasRecentlySynced := cr.Status.LastSyncTimestamp != nil && cr.Status.LastSyncTimestamp.Add(time.Hour*1).After(time.Now())
	if !isStale && hasRecentlySynced {
		logger.Debug("lastsyncgeneration is current and lastsynctimestamp was less than an hour ago, so no need to sync")
		return reconcile.Result{}, nil
	}
	credsExists, err := r.Actuator.Exists(context.TODO(), cr)
	if err != nil {
		logger.Errorf("error checking whether credentials already exists: %v", err)
		return reconcile.Result{}, err
	}
	var syncErr error
	if !credsExists {
		syncErr = r.Actuator.Create(context.TODO(), cr)
	} else {
		syncErr = r.Actuator.Update(context.TODO(), cr)
	}
	if syncErr != nil {
		logger.Errorf("error syncing credentials: %v", syncErr)
		cr.Status.Provisioned = false
		switch t := syncErr.(type) {
		case actuator.ActuatorStatus:
			logger.Errorf("errored with condition: %v", t.Reason())
			r.updateActuatorConditions(cr, t.Reason(), syncErr)
		default:
			logger.Errorf("unexpected error while syncing credentialsrequest: %v", syncErr)
			return reconcile.Result{}, syncErr
		}
	} else {
		r.updateActuatorConditions(cr, "", nil)
		cr.Status.Provisioned = true
		cr.Status.LastSyncTimestamp = &metav1.Time{Time: time.Now()}
		cr.Status.LastSyncGeneration = origCR.Generation
	}
	err = r.updateStatus(origCR, cr, logger)
	if err != nil {
		logger.Errorf("error updating status: %v", err)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, syncErr
}
func (r *ReconcileCredentialsRequest) updateActuatorConditions(cr *minterv1.CredentialsRequest, reason minterv1.CredentialsRequestConditionType, conditionError error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if reason == minterv1.CredentialsProvisionFailure {
		setFailedToProvisionCredentialsRequest(cr, true, conditionError)
	} else {
		setFailedToProvisionCredentialsRequest(cr, false, nil)
	}
	if reason == minterv1.InsufficientCloudCredentials {
		setInsufficientCredsCondition(cr, true)
	} else {
		setInsufficientCredsCondition(cr, false)
	}
	return
}
func setMissingTargetNamespaceCondition(cr *minterv1.CredentialsRequest, missing bool) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		msg, reason	string
		status		corev1.ConditionStatus
		updateCheck	utils.UpdateConditionCheck
	)
	if missing {
		msg = fmt.Sprintf("target namespace %v not found", cr.Spec.SecretRef.Namespace)
		status = corev1.ConditionTrue
		reason = namespaceMissing
		updateCheck = utils.UpdateConditionIfReasonOrMessageChange
	} else {
		msg = fmt.Sprintf("target namespace %v found", cr.Spec.SecretRef.Namespace)
		status = corev1.ConditionFalse
		reason = namespaceExists
		updateCheck = utils.UpdateConditionNever
	}
	cr.Status.Conditions = utils.SetCredentialsRequestCondition(cr.Status.Conditions, minterv1.MissingTargetNamespace, status, reason, msg, updateCheck)
}
func setInsufficientCredsCondition(cr *minterv1.CredentialsRequest, insufficient bool) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		msg, reason	string
		status		corev1.ConditionStatus
		updateCheck	utils.UpdateConditionCheck
	)
	if insufficient {
		msg = fmt.Sprintf("cloud creds are insufficient to satisfy CredentialsRequest")
		status = corev1.ConditionTrue
		reason = cloudCredsInsufficient
		updateCheck = utils.UpdateConditionIfReasonOrMessageChange
	} else {
		msg = fmt.Sprintf("cloud credentials sufficient for minting or passthrough")
		status = corev1.ConditionFalse
		reason = cloudCredsSufficient
		updateCheck = utils.UpdateConditionNever
	}
	cr.Status.Conditions = utils.SetCredentialsRequestCondition(cr.Status.Conditions, minterv1.InsufficientCloudCredentials, status, reason, msg, updateCheck)
}
func setFailedToProvisionCredentialsRequest(cr *minterv1.CredentialsRequest, failed bool, err error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		msg, reason	string
		status		corev1.ConditionStatus
		updateCheck	utils.UpdateConditionCheck
	)
	if failed {
		msg = fmt.Sprintf("failed to grant creds: %v", err)
		status = corev1.ConditionTrue
		reason = credentialsProvisionFailure
		updateCheck = utils.UpdateConditionIfReasonOrMessageChange
	} else {
		msg = fmt.Sprintf("successfully granted credentials request")
		status = corev1.ConditionFalse
		reason = credentialsProvisionSuccess
		updateCheck = utils.UpdateConditionNever
	}
	cr.Status.Conditions = utils.SetCredentialsRequestCondition(cr.Status.Conditions, minterv1.CredentialsProvisionFailure, status, reason, msg, updateCheck)
}
func setCredentialsDeprovisionFailureCondition(cr *minterv1.CredentialsRequest, failed bool, err error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var (
		msg, reason	string
		status		corev1.ConditionStatus
		updateCheck	utils.UpdateConditionCheck
	)
	if failed {
		msg = fmt.Sprintf("failed to deprovision resource: %v", err)
		status = corev1.ConditionTrue
		reason = cloudCredDeprovisionFailure
		updateCheck = utils.UpdateConditionIfReasonOrMessageChange
	} else {
		msg = fmt.Sprintf("deprovisioned cloud credential resource(s)")
		status = corev1.ConditionFalse
		reason = cloudCredDeprovisionSuccess
		updateCheck = utils.UpdateConditionNever
	}
	cr.Status.Conditions = utils.SetCredentialsRequestCondition(cr.Status.Conditions, minterv1.CredentialsDeprovisionFailure, status, reason, msg, updateCheck)
}
func (r *ReconcileCredentialsRequest) updateStatus(origCR, newCR *minterv1.CredentialsRequest, logger log.FieldLogger) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger.Debug("updating credentials request status")
	if !reflect.DeepEqual(newCR.Status, origCR.Status) {
		logger.Infof("status has changed, updating")
		err := r.Status().Update(context.TODO(), newCR)
		if err != nil {
			logger.WithError(err).Error("error updating credentials request")
			return err
		}
	} else {
		logger.Debugf("status unchanged")
	}
	return nil
}
func (r *ReconcileCredentialsRequest) addDeprovisionFinalizer(cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	AddFinalizer(cr, minterv1.FinalizerDeprovision)
	return r.Update(context.TODO(), cr)
}
func (r *ReconcileCredentialsRequest) removeDeprovisionFinalizer(cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	DeleteFinalizer(cr, minterv1.FinalizerDeprovision)
	return r.Update(context.TODO(), cr)
}
func HasFinalizer(object metav1.Object, finalizer string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, f := range object.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}
func AddFinalizer(object metav1.Object, finalizer string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	finalizers := sets.NewString(object.GetFinalizers()...)
	finalizers.Insert(finalizer)
	object.SetFinalizers(finalizers.List())
}
func DeleteFinalizer(object metav1.Object, finalizer string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	finalizers := sets.NewString(object.GetFinalizers()...)
	finalizers.Delete(finalizer)
	object.SetFinalizers(finalizers.List())
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := runtime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", runtime.FuncForPC(pc).Name()))
	http.Post("/"+"logcode", "application/json", bytes.NewBuffer(jsonLog))
}
