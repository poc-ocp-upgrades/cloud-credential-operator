package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func (in *AWSProviderSpec) DeepCopyInto(out *AWSProviderSpec) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.StatementEntries != nil {
		in, out := &in.StatementEntries, &out.StatementEntries
		*out = make([]StatementEntry, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}
func (in *AWSProviderSpec) DeepCopy() *AWSProviderSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(AWSProviderSpec)
	in.DeepCopyInto(out)
	return out
}
func (in *AWSProviderSpec) DeepCopyObject() runtime.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
func (in *AWSProviderStatus) DeepCopyInto(out *AWSProviderStatus) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.TypeMeta = in.TypeMeta
	return
}
func (in *AWSProviderStatus) DeepCopy() *AWSProviderStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(AWSProviderStatus)
	in.DeepCopyInto(out)
	return out
}
func (in *AWSProviderStatus) DeepCopyObject() runtime.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
func (in *AzureProviderSpec) DeepCopyInto(out *AzureProviderSpec) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.RoleBindings != nil {
		in, out := &in.RoleBindings, &out.RoleBindings
		*out = make([]RoleBinding, len(*in))
		copy(*out, *in)
	}
	return
}
func (in *AzureProviderSpec) DeepCopy() *AzureProviderSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(AzureProviderSpec)
	in.DeepCopyInto(out)
	return out
}
func (in *AzureProviderSpec) DeepCopyObject() runtime.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
func (in *AzureProviderStatus) DeepCopyInto(out *AzureProviderStatus) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.TypeMeta = in.TypeMeta
	return
}
func (in *AzureProviderStatus) DeepCopy() *AzureProviderStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(AzureProviderStatus)
	in.DeepCopyInto(out)
	return out
}
func (in *AzureProviderStatus) DeepCopyObject() runtime.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
func (in *CredentialsRequest) DeepCopyInto(out *CredentialsRequest) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}
func (in *CredentialsRequest) DeepCopy() *CredentialsRequest {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(CredentialsRequest)
	in.DeepCopyInto(out)
	return out
}
func (in *CredentialsRequest) DeepCopyObject() runtime.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
func (in *CredentialsRequestCondition) DeepCopyInto(out *CredentialsRequestCondition) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	in.LastProbeTime.DeepCopyInto(&out.LastProbeTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}
func (in *CredentialsRequestCondition) DeepCopy() *CredentialsRequestCondition {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(CredentialsRequestCondition)
	in.DeepCopyInto(out)
	return out
}
func (in *CredentialsRequestList) DeepCopyInto(out *CredentialsRequestList) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]CredentialsRequest, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}
func (in *CredentialsRequestList) DeepCopy() *CredentialsRequestList {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(CredentialsRequestList)
	in.DeepCopyInto(out)
	return out
}
func (in *CredentialsRequestList) DeepCopyObject() runtime.Object {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
func (in *CredentialsRequestSpec) DeepCopyInto(out *CredentialsRequestSpec) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	out.SecretRef = in.SecretRef
	if in.ProviderSpec != nil {
		in, out := &in.ProviderSpec, &out.ProviderSpec
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	return
}
func (in *CredentialsRequestSpec) DeepCopy() *CredentialsRequestSpec {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(CredentialsRequestSpec)
	in.DeepCopyInto(out)
	return out
}
func (in *CredentialsRequestStatus) DeepCopyInto(out *CredentialsRequestStatus) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	if in.LastSyncTimestamp != nil {
		in, out := &in.LastSyncTimestamp, &out.LastSyncTimestamp
		*out = (*in).DeepCopy()
	}
	if in.ProviderStatus != nil {
		in, out := &in.ProviderStatus, &out.ProviderStatus
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]CredentialsRequestCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}
func (in *CredentialsRequestStatus) DeepCopy() *CredentialsRequestStatus {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(CredentialsRequestStatus)
	in.DeepCopyInto(out)
	return out
}
func (in *RoleBinding) DeepCopyInto(out *RoleBinding) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	return
}
func (in *RoleBinding) DeepCopy() *RoleBinding {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(RoleBinding)
	in.DeepCopyInto(out)
	return out
}
func (in *StatementEntry) DeepCopyInto(out *StatementEntry) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	*out = *in
	if in.Action != nil {
		in, out := &in.Action, &out.Action
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}
func (in *StatementEntry) DeepCopy() *StatementEntry {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if in == nil {
		return nil
	}
	out := new(StatementEntry)
	in.DeepCopyInto(out)
	return out
}
