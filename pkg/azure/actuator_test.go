package azure_test

import (
	"testing"
	"github.com/openshift/cloud-credential-operator/pkg/azure"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAnnotations(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var tests = []struct {
		name		string
		in		corev1.Secret
		errRegexp	string
	}{{"TestValidSecretAnnotation", validRootSecret, ""}, {"TestBadSecretAnnotation", rootSecretBadAnnotation, "invalid mode"}, {"TestMissingSecretAnnotation", rootSecretNoAnnotation, "cannot proceed without cloud cred secret annotation.*"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fake.NewFakeClient(&tt.in, &validSecret)
			actuator, err := azure.NewActuator(f)
			if err != nil {
				assert.Regexp(t, tt.errRegexp, err)
				assert.Nil(t, actuator)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, actuator)
		})
	}
}
