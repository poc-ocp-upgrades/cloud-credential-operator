package actuator

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestGenerateUserName(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	tests := []struct {
		name		string
		clusterName	string
		credentialName	string
		expectedPrefix	string
		expectedError	bool
	}{{name: "max size no truncating required", clusterName: "20charclustername111", credentialName: "openshift-cluster-ingress111111111111", expectedPrefix: "20charclustername111-openshift-cluster-ingress111111111111-"}, {name: "credential name truncated to 37 chars", clusterName: "shortcluster", credentialName: "openshift-cluster-ingress111111111111333333333333333", expectedPrefix: "shortcluster-openshift-cluster-ingress111111111111-"}, {name: "cluster name truncated to 20 chars", clusterName: "longclustername1111137492374923874928347928374", credentialName: "openshift-cluster-ingress", expectedPrefix: "longclustername11111-openshift-cluster-ingress-"}, {name: "empty credential name", clusterName: "shortcluster", credentialName: "", expectedError: true}, {name: "empty infra name", clusterName: "", credentialName: "something", expectedPrefix: "something-"}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userName, err := generateUserName(test.clusterName, test.credentialName)
			if err != nil && !test.expectedError {
				t.Errorf("unexpected error: %v", err)
			} else if err == nil {
				if test.expectedError {
					t.Error("no error returned")
				} else {
					t.Logf("userName: %s, length=%d", userName, len(userName))
					assert.True(t, len(userName) <= 64)
					if test.expectedPrefix != "" {
						assert.True(t, strings.HasPrefix(userName, test.expectedPrefix), "username prefix does not match")
						assert.Equal(t, len(test.expectedPrefix)+5, len(userName), "username length does not match a 5 char random suffix")
					}
				}
			}
		})
	}
}
