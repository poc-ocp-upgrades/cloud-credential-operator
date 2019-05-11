package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	FinalizerDeprovision			string	= "cloudcredential.openshift.io/deprovision"
	AnnotationCredentialsRequest	string	= "cloudcredential.openshift.io/credentials-request"
	AnnotationAWSPolicyLastApplied	string	= "cloudcredential.openshift.io/aws-policy-last-applied"
)

type CredentialsRequestSpec struct {
	SecretRef		corev1.ObjectReference	`json:"secretRef"`
	ProviderSpec	*runtime.RawExtension	`json:"providerSpec,omitempty"`
}
type CredentialsRequestStatus struct {
	Provisioned			bool							`json:"provisioned"`
	LastSyncTimestamp	*metav1.Time					`json:"lastSyncTimestamp,omitempty"`
	LastSyncGeneration	int64							`json:"lastSyncGeneration"`
	ProviderStatus		*runtime.RawExtension			`json:"providerStatus,omitempty"`
	Conditions			[]CredentialsRequestCondition	`json:"conditions,omitempty"`
}
type CredentialsRequest struct {
	metav1.TypeMeta		`json:",inline"`
	metav1.ObjectMeta	`json:"metadata,omitempty"`
	Spec				CredentialsRequestSpec		`json:"spec"`
	Status				CredentialsRequestStatus	`json:"status,omitempty"`
}
type CredentialsRequestList struct {
	metav1.TypeMeta	`json:",inline"`
	metav1.ListMeta	`json:"metadata,omitempty"`
	Items			[]CredentialsRequest	`json:"items"`
}
type CredentialsRequestCondition struct {
	Type				CredentialsRequestConditionType	`json:"type"`
	Status				corev1.ConditionStatus			`json:"status"`
	LastProbeTime		metav1.Time						`json:"lastProbeTime,omitempty"`
	LastTransitionTime	metav1.Time						`json:"lastTransitionTime,omitempty"`
	Reason				string							`json:"reason,omitempty"`
	Message				string							`json:"message,omitempty"`
}
type CredentialsRequestConditionType string

const (
	InsufficientCloudCredentials	CredentialsRequestConditionType	= "InsufficientCloudCreds"
	MissingTargetNamespace			CredentialsRequestConditionType	= "MissingTargetNamespace"
	CredentialsProvisionFailure		CredentialsRequestConditionType	= "CredentialsProvisionFailure"
	CredentialsDeprovisionFailure	CredentialsRequestConditionType	= "CredentialsDeprovisionFailure"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	SchemeBuilder.Register(&CredentialsRequest{}, &CredentialsRequestList{}, &AWSProviderStatus{}, &AWSProviderSpec{})
}
