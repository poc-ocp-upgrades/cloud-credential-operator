package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AzureProviderSpec struct {
	metav1.TypeMeta	`json:",inline"`
	RoleBindings	[]RoleBinding	`json:"roleBindings"`
}
type RoleBinding struct {
	Role	string	`json:"role"`
	Scope	string	`json:"scope"`
}
type AzureProviderStatus struct {
	metav1.TypeMeta		`json:",inline"`
	ServicePrincipalName	string	`json:"name"`
	AppID			string	`json:"appID"`
}
