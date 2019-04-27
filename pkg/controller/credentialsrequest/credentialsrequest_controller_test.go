package credentialsrequest

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"
	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"github.com/openshift/cloud-credential-operator/pkg/apis"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	minteraws "github.com/openshift/cloud-credential-operator/pkg/aws"
	"github.com/openshift/cloud-credential-operator/pkg/aws/actuator"
	mockaws "github.com/openshift/cloud-credential-operator/pkg/aws/mock"
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
)

var c client.Client

const (
	openshiftClusterIDKey = "openshiftClusterID"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	log.SetLevel(log.DebugLevel)
}

type ExpectedCondition struct {
	conditionType	minterv1.CredentialsRequestConditionType
	reason		string
	status		corev1.ConditionStatus
}
type ExpectedCOCondition struct {
	conditionType	configv1.ClusterStatusConditionType
	reason		string
	status		corev1.ConditionStatus
}

func TestCredentialsRequestReconcile(t *testing.T) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	apis.AddToScheme(scheme.Scheme)
	configv1.Install(scheme.Scheme)
	getCR := func(c client.Client) *minterv1.CredentialsRequest {
		cr := &minterv1.CredentialsRequest{}
		err := c.Get(context.TODO(), client.ObjectKey{Name: testCRName, Namespace: testNamespace}, cr)
		if err == nil {
			return cr
		}
		return nil
	}
	getSecret := func(c client.Client) *corev1.Secret {
		secret := &corev1.Secret{}
		err := c.Get(context.TODO(), client.ObjectKey{Name: testSecretName, Namespace: testSecretNamespace}, secret)
		if err == nil {
			return secret
		}
		return nil
	}
	getClusterOperator := func(c client.Client) *configv1.ClusterOperator {
		co := &configv1.ClusterOperator{ObjectMeta: metav1.ObjectMeta{Name: cloudCredClusterOperator}}
		err := c.Get(context.TODO(), types.NamespacedName{Name: co.Name}, co)
		if err == nil {
			return co
		}
		return nil
	}
	codec, err := minterv1.NewCodec()
	if err != nil {
		fmt.Printf("error creating codec: %v", err)
		t.FailNow()
		return
	}
	tests := []struct {
		name			string
		existing		[]runtime.Object
		expectErr		bool
		mockRootAWSClient	func(mockCtrl *gomock.Controller) *mockaws.MockClient
		mockReadAWSClient	func(mockCtrl *gomock.Controller) *mockaws.MockClient
		mockSecretAWSClient	func(mockCtrl *gomock.Controller) *mockaws.MockClient
		validate		func(client.Client, *testing.T)
		expectedConditions	[]ExpectedCondition
		expectedCOConditions	[]ExpectedCOCondition
	}{{name: "add finalizer", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), func() *minterv1.CredentialsRequest {
		cr := testCredentialsRequest(t)
		cr.ObjectMeta.Finalizers = []string{}
		return cr
	}(), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		cr := getCR(c)
		if cr == nil || !HasFinalizer(cr, minterv1.FinalizerDeprovision) {
			t.Errorf("did not get expected finalizer")
		}
		assert.False(t, cr.Status.Provisioned)
	}}, {name: "new credential", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockCreateUser(mockAWSClient)
		mockPutUserPolicy(mockAWSClient)
		mockCreateAccessKey(mockAWSClient, testAWSAccessKeyID, testAWSSecretAccessKey)
		mockTagUser(mockAWSClient)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUserNotFound(mockAWSClient)
		mockGetUserPolicyMissing(mockAWSClient)
		mockListAccessKeysEmpty(mockAWSClient)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
		assert.Equal(t, int64(testCRGeneration), int64(cr.Status.LastSyncGeneration))
		assert.NotNil(t, cr.Status.LastSyncTimestamp)
	}, expectedCOConditions: []ExpectedCOCondition{{conditionType: configv1.OperatorAvailable, status: corev1.ConditionTrue}, {conditionType: configv1.OperatorProgressing, status: corev1.ConditionFalse}, {conditionType: configv1.OperatorFailing, status: corev1.ConditionFalse}}}, {name: "new credential cluster has no infra name", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testClusterVersion(), testInfrastructure("")}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockCreateUser(mockAWSClient)
		mockPutUserPolicy(mockAWSClient)
		mockCreateAccessKey(mockAWSClient, testAWSAccessKeyID, testAWSSecretAccessKey)
		mockTagUserLegacy(mockAWSClient)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUserNotFound(mockAWSClient)
		mockGetUserPolicyMissing(mockAWSClient)
		mockListAccessKeysEmpty(mockAWSClient)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
		assert.Equal(t, int64(testCRGeneration), int64(cr.Status.LastSyncGeneration))
		assert.NotNil(t, cr.Status.LastSyncTimestamp)
	}, expectedCOConditions: []ExpectedCOCondition{{conditionType: configv1.OperatorAvailable, status: corev1.ConditionTrue}, {conditionType: configv1.OperatorProgressing, status: corev1.ConditionFalse}, {conditionType: configv1.OperatorFailing, status: corev1.ConditionFalse}}}, {name: "new credential no read-only creds available", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockCreateUser(mockAWSClient)
		mockPutUserPolicy(mockAWSClient)
		mockCreateAccessKey(mockAWSClient, testAWSAccessKeyID, testAWSSecretAccessKey)
		mockTagUser(mockAWSClient)
		mockGetUserNotFound(mockAWSClient)
		mockGetUserPolicyMissing(mockAWSClient)
		mockListAccessKeysEmpty(mockAWSClient)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}, expectedCOConditions: []ExpectedCOCondition{{conditionType: configv1.OperatorAvailable, status: corev1.ConditionTrue}, {conditionType: configv1.OperatorProgressing, status: corev1.ConditionFalse}, {conditionType: configv1.OperatorFailing, status: corev1.ConditionFalse}}}, {name: "new credential no root creds available", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		assert.Nil(t, targetSecret)
		cr := getCR(c)
		assert.False(t, cr.Status.Provisioned)
	}, expectErr: true, expectedCOConditions: []ExpectedCOCondition{{conditionType: configv1.OperatorAvailable, status: corev1.ConditionTrue}, {conditionType: configv1.OperatorProgressing, status: corev1.ConditionTrue}}}, {name: "cred and secret exist user tagged", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testSecretNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUser(mockAWSClient)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		mockGetUserPolicy(mockAWSClient, testPolicy1)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}}, {name: "cred and secret exist user missing tag", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testSecretNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockTagUser(mockAWSClient)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUserUntagged(mockAWSClient)
		mockGetUserPolicy(mockAWSClient, testPolicy1)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		return mockAWSClient
	}, mockSecretAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}}, {name: "cred and secret exist no root creds", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testSecretNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUser(mockAWSClient)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		mockGetUserPolicy(mockAWSClient, testPolicy1)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}, expectedCOConditions: []ExpectedCOCondition{{conditionType: configv1.OperatorAvailable, status: corev1.ConditionTrue}, {conditionType: configv1.OperatorProgressing, status: corev1.ConditionFalse}, {conditionType: configv1.OperatorFailing, status: corev1.ConditionFalse}}}, {name: "cred missing access key exists", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockCreateAccessKey(mockAWSClient, testAWSAccessKeyID2, testAWSSecretAccessKey2)
		mockDeleteAccessKey(mockAWSClient, testAWSAccessKeyID)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUser(mockAWSClient)
		mockGetUserPolicy(mockAWSClient, testPolicy1)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID2, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey2, string(targetSecret.Data["aws_secret_access_key"]))
			assert.Equal(t, fmt.Sprintf("%s/%s", testNamespace, testCRName), targetSecret.Annotations[minterv1.AnnotationCredentialsRequest])
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}}, {name: "cred exists access key missing", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testSecretNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockCreateAccessKey(mockAWSClient, testAWSAccessKeyID2, testAWSSecretAccessKey2)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUser(mockAWSClient)
		mockListAccessKeysEmpty(mockAWSClient)
		mockGetUserPolicy(mockAWSClient, testPolicy1)
		mockListAccessKeysEmpty(mockAWSClient)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testAWSAccessKeyID2, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testAWSSecretAccessKey2, string(targetSecret.Data["aws_secret_access_key"]))
			assert.Equal(t, fmt.Sprintf("%s/%s", testNamespace, testCRName), targetSecret.Annotations[minterv1.AnnotationCredentialsRequest])
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}}, {name: "cred deletion", existing: []runtime.Object{createTestNamespace(testNamespace), createTestNamespace(testSecretNamespace), testCredentialsRequestWithDeletionTimestamp(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testSecretNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey), testClusterVersion(), testInfrastructure(testInfraName)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		mockDeleteUser(mockAWSClient)
		mockDeleteUserPolicy(mockAWSClient)
		mockDeleteAccessKey(mockAWSClient, testAWSAccessKeyID)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		assert.Nil(t, targetSecret)
	}, expectedCOConditions: []ExpectedCOCondition{{conditionType: configv1.OperatorAvailable, status: corev1.ConditionTrue}, {conditionType: configv1.OperatorProgressing, status: corev1.ConditionFalse}, {conditionType: configv1.OperatorFailing, status: corev1.ConditionFalse}}}, {name: "new passthrough credential", existing: []runtime.Object{createTestNamespace(testNamespace), testInfrastructure(testInfraName), createTestNamespace(testSecretNamespace), testPassthroughCredentialsRequest(t), testPassthroughAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		targetSecret := getSecret(c)
		if assert.NotNil(t, targetSecret) {
			assert.Equal(t, testRootAWSAccessKeyID, string(targetSecret.Data["aws_access_key_id"]))
			assert.Equal(t, testRootAWSSecretAccessKey, string(targetSecret.Data["aws_secret_access_key"]))
		}
		cr := getCR(c)
		assert.True(t, cr.Status.Provisioned)
	}}, {name: "passthrough cred deletion", existing: []runtime.Object{createTestNamespace(testNamespace), testInfrastructure(testInfraName), createTestNamespace(testSecretNamespace), testPassthroughCredentialsRequestWithDeletionTimestamp(t), testPassthroughAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testSecretNamespace, testSecretName, testRootAWSAccessKeyID, testRootAWSSecretAccessKey)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}}, {name: "no namespace condition", existing: []runtime.Object{testInfrastructure(testInfraName), testCredentialsRequest(t)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, expectedConditions: []ExpectedCondition{{conditionType: minterv1.MissingTargetNamespace, reason: "NamespaceMissing", status: corev1.ConditionTrue}}}, {name: "insufficient creds", existing: []runtime.Object{testInfrastructure(testInfraName), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testInsufficientAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, expectErr: true, expectedConditions: []ExpectedCondition{{conditionType: minterv1.InsufficientCloudCredentials, reason: "CloudCredsInsufficient", status: corev1.ConditionTrue}}}, {name: "failed to mint condition", existing: []runtime.Object{testInfrastructure(testInfraName), createTestNamespace(testSecretNamespace), testCredentialsRequest(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testClusterVersion()}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockFailedGetUser(mockAWSClient)
		return mockAWSClient
	}, expectErr: true, expectedConditions: []ExpectedCondition{{conditionType: minterv1.CredentialsProvisionFailure, reason: "CredentialsProvisionFailure", status: corev1.ConditionTrue}}}, {name: "cred deletion failure condition", existing: []runtime.Object{createTestNamespace(testSecretNamespace), testInfrastructure(testInfraName), testCredentialsRequestWithDeletionTimestamp(t), testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey), testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey), testAWSCredsSecret(testNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey), testClusterVersion()}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		mockDeleteUserPolicy(mockAWSClient)
		mockDeleteAccessKey(mockAWSClient, testAWSAccessKeyID)
		mockDeleteUserFailure(mockAWSClient)
		return mockAWSClient
	}, expectErr: true, expectedConditions: []ExpectedCondition{{conditionType: minterv1.CredentialsDeprovisionFailure, reason: "CloudCredDeprovisionFailure", status: corev1.ConditionTrue}}}, {name: "skip AWS if recently synced", existing: []runtime.Object{createTestNamespace(testNamespace), testInfrastructure(testInfraName), createTestNamespace(testSecretNamespace), testCredentialsRequestWithRecentLastSync(t)}, mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		cr := getCR(c)
		assert.Equal(t, testTwentyMinuteOldTimestamp.Unix(), cr.Status.LastSyncTimestamp.Time.Unix())
	}}, {name: "recently synced but modified", existing: func() []runtime.Object {
		objects := []runtime.Object{}
		objects = append(objects, createTestNamespace(testNamespace))
		objects = append(objects, createTestNamespace(testSecretNamespace))
		cr := testCredentialsRequestWithRecentLastSync(t)
		cr.Generation = cr.Generation + 1
		objects = append(objects, cr)
		objects = append(objects, testAWSCredsSecret("kube-system", "aws-creds", testRootAWSAccessKeyID, testRootAWSSecretAccessKey))
		objects = append(objects, testAWSCredsSecret("openshift-cloud-credential-operator", "cloud-credential-operator-iam-ro-creds", testReadAWSAccessKeyID, testReadAWSSecretAccessKey))
		objects = append(objects, testAWSCredsSecret(testSecretNamespace, testSecretName, testAWSAccessKeyID, testAWSSecretAccessKey))
		objects = append(objects, testClusterVersion())
		objects = append(objects, testInfrastructure(testInfraName))
		return objects
	}(), mockRootAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		return mockAWSClient
	}, mockReadAWSClient: func(mockCtrl *gomock.Controller) *mockaws.MockClient {
		mockAWSClient := mockaws.NewMockClient(mockCtrl)
		mockGetUser(mockAWSClient)
		mockListAccessKeys(mockAWSClient, testAWSAccessKeyID)
		mockGetUserPolicy(mockAWSClient, testPolicy1)
		return mockAWSClient
	}, validate: func(c client.Client, t *testing.T) {
		cr := getCR(c)
		assert.NotEqual(t, testTwentyMinuteOldTimestamp.Unix(), cr.Status.LastSyncTimestamp.Time.Unix())
	}}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockRootAWSClient := test.mockRootAWSClient(mockCtrl)
			mockReadAWSClient := mockaws.NewMockClient(mockCtrl)
			if test.mockReadAWSClient != nil {
				mockReadAWSClient = test.mockReadAWSClient(mockCtrl)
			}
			mockSecretAWSClient := mockaws.NewMockClient(mockCtrl)
			if test.mockSecretAWSClient != nil {
				mockSecretAWSClient = test.mockSecretAWSClient(mockCtrl)
			}
			fakeClient := fake.NewFakeClient(test.existing...)
			rcr := &ReconcileCredentialsRequest{Client: fakeClient, Actuator: &actuator.AWSActuator{Client: fakeClient, Codec: codec, Scheme: scheme.Scheme, AWSClientBuilder: func(accessKeyID, secretAccessKey []byte, infraName string) (minteraws.Client, error) {
				if string(accessKeyID) == testRootAWSAccessKeyID {
					return mockRootAWSClient, nil
				} else if string(accessKeyID) == testAWSAccessKeyID {
					return mockSecretAWSClient, nil
				} else {
					return mockReadAWSClient, nil
				}
			}}}
			_, err = rcr.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: testCRName, Namespace: testNamespace}})
			if test.validate != nil {
				test.validate(fakeClient, t)
			}
			if err != nil && !test.expectErr {
				t.Errorf("Unexpected error: %v", err)
			}
			if err == nil && test.expectErr {
				t.Errorf("Expected error but got none")
			}
			cr := getCR(fakeClient)
			for _, condition := range test.expectedConditions {
				foundCondition := utils.FindCredentialsRequestCondition(cr.Status.Conditions, condition.conditionType)
				assert.NotNil(t, foundCondition)
				assert.Exactly(t, condition.status, foundCondition.Status)
				assert.Exactly(t, condition.reason, foundCondition.Reason)
			}
			for _, condition := range test.expectedCOConditions {
				co := getClusterOperator(fakeClient)
				assert.NotNil(t, co)
				foundCondition := findClusterOperatorCondition(co.Status.Conditions, condition.conditionType)
				assert.NotNil(t, foundCondition)
				assert.Equal(t, string(condition.status), string(foundCondition.Status), "condition %s had unexpected status", condition.conditionType)
				if condition.reason != "" {
					assert.Exactly(t, condition.reason, foundCondition.Reason)
				}
			}
		})
	}
}

const (
	testCRGeneration		= 1
	testCRName			= "openshift-component-a"
	testNamespace			= "openshift-cloud-credential-operator"
	testClusterName			= "testcluster"
	testClusterID			= "e415fe1c-f894-11e8-8eb2-f2801f1b9fd1"
	testInfraName			= "testcluster-abc123"
	testSecretName			= "test-secret"
	testSecretNamespace		= "myproject"
	testAWSUser			= "mycluster-test-aws-user"
	testAWSARN			= "some:fake:ARN:1234"
	testAWSUserID			= "FAKEAWSUSERID"
	testAWSAccessKeyID		= "FAKEAWSACCESSKEYID"
	testAWSAccessKeyID2		= "FAKEAWSACCESSKEYID2"
	testAWSSecretAccessKey		= "KEEPITSECRET"
	testAWSSecretAccessKey2		= "KEEPITSECRET2"
	testRootAWSAccessKeyID		= "rootaccesskey"
	testRootAWSSecretAccessKey	= "rootsecretkey"
	testReadAWSAccessKeyID		= "readaccesskey"
	testReadAWSSecretAccessKey	= "readsecretkey"
)

var (
	testPolicy1			= fmt.Sprintf("{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"iam:GetUser\",\"iam:GetUserPolicy\",\"iam:ListAccessKeys\"],\"Resource\":\"*\"},{\"Effect\":\"Allow\",\"Action\":[\"iam:GetUser\"],\"Resource\":\"%s\"}]}", testAWSARN)
	testTwentyMinuteOldTimestamp	= time.Now().Add(-20 * time.Minute)
)

func testPassthroughCredentialsRequestWithDeletionTimestamp(t *testing.T) *minterv1.CredentialsRequest {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	cr := testPassthroughCredentialsRequest(t)
	now := metav1.Now()
	cr.DeletionTimestamp = &now
	return cr
}
func testCredentialsRequestWithRecentLastSync(t *testing.T) *minterv1.CredentialsRequest {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	cr := testCredentialsRequest(t)
	cr.Status.LastSyncGeneration = cr.Generation
	cr.Status.LastSyncTimestamp = &metav1.Time{Time: testTwentyMinuteOldTimestamp}
	return cr
}
func testCredentialsRequestWithDeletionTimestamp(t *testing.T) *minterv1.CredentialsRequest {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	cr := testCredentialsRequest(t)
	now := metav1.Now()
	cr.DeletionTimestamp = &now
	cr.Status.Provisioned = true
	return cr
}
func testPassthroughCredentialsRequest(t *testing.T) *minterv1.CredentialsRequest {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	codec, err := minterv1.NewCodec()
	if err != nil {
		t.Logf("error creating new codec: %v", err)
		t.FailNow()
		return nil
	}
	awsProvSpec, err := codec.EncodeProviderSpec(&minterv1.AWSProviderSpec{StatementEntries: []minterv1.StatementEntry{{Effect: "Allow", Action: []string{"iam:GetUser", "iam:GetUserPolicy", "iam:ListAccessKeys"}, Resource: "*"}}})
	if err != nil {
		t.Logf("error encoding: %v", err)
		t.FailNow()
		return nil
	}
	return &minterv1.CredentialsRequest{ObjectMeta: metav1.ObjectMeta{Name: testCRName, Namespace: testNamespace, Finalizers: []string{minterv1.FinalizerDeprovision}, UID: types.UID("1234"), Annotations: map[string]string{}, Generation: testCRGeneration}, Spec: minterv1.CredentialsRequestSpec{SecretRef: corev1.ObjectReference{Name: testSecretName, Namespace: testSecretNamespace}, ProviderSpec: awsProvSpec}}
}
func testCredentialsRequest(t *testing.T) *minterv1.CredentialsRequest {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	cr := testPassthroughCredentialsRequest(t)
	codec, err := minterv1.NewCodec()
	if err != nil {
		t.Logf("error creating new codec: %v", err)
		t.FailNow()
		return nil
	}
	awsStatus, err := codec.EncodeProviderStatus(&minterv1.AWSProviderStatus{User: testAWSUser})
	if err != nil {
		t.Logf("error encoding: %v", err)
		t.FailNow()
		return nil
	}
	cr.Status.ProviderStatus = awsStatus
	return cr
}
func createTestNamespace(namespace string) *corev1.Namespace {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
}
func testInsufficientAWSCredsSecret(namespace, name, accessKeyID, secretAccessKey string) *corev1.Secret {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	s := testAWSCredsSecret(namespace, name, accessKeyID, secretAccessKey)
	s.Annotations[secretannotator.AnnotationKey] = secretannotator.InsufficientAnnotation
	return s
}
func testPassthroughAWSCredsSecret(namespace, name, accessKeyID, secretAccessKey string) *corev1.Secret {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	s := testAWSCredsSecret(namespace, name, accessKeyID, secretAccessKey)
	s.Annotations[secretannotator.AnnotationKey] = secretannotator.PassthroughAnnotation
	return s
}
func testAWSCredsSecret(namespace, name, accessKeyID, secretAccessKey string) *corev1.Secret {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Annotations: map[string]string{secretannotator.AnnotationKey: secretannotator.MintAnnotation}}, Data: map[string][]byte{"aws_access_key_id": []byte(accessKeyID), "aws_secret_access_key": []byte(secretAccessKey)}}
	return s
}
func genericAWSError() error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return awserr.New("GenericFailure", "An error besides NotFound", fmt.Errorf("Just a generic AWS error for test purposes"))
}
func mockFailedGetUser(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().GetUser(gomock.Any()).Return(nil, genericAWSError()).AnyTimes()
}
func mockGetUserNotFound(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().GetUser(gomock.Any()).Return(nil, awserr.New(iam.ErrCodeNoSuchEntityException, "no such entity", nil)).AnyTimes()
}
func mockGetUser(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().GetUser(gomock.Any()).Return(&iam.GetUserOutput{User: &iam.User{UserId: aws.String(testAWSUserID), UserName: aws.String(testAWSUser), Arn: aws.String(testAWSARN), Tags: []*iam.Tag{{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", testInfraName)), Value: aws.String("owned")}}}}, nil).AnyTimes()
}
func mockGetUserUntagged(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().GetUser(gomock.Any()).Return(&iam.GetUserOutput{User: &iam.User{UserId: aws.String(testAWSUserID), UserName: aws.String(testAWSUser), Arn: aws.String(testAWSARN)}}, nil).AnyTimes()
}
func mockDeleteUserFailure(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().DeleteUser(gomock.Any()).Return(nil, genericAWSError())
}
func mockDeleteUser(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().DeleteUser(gomock.Any()).Return(&iam.DeleteUserOutput{}, nil)
}
func mockDeleteUserPolicy(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().DeleteUserPolicy(gomock.Any()).Return(&iam.DeleteUserPolicyOutput{}, nil)
}
func mockListAccessKeysEmpty(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(testAWSUser)}).Return(&iam.ListAccessKeysOutput{AccessKeyMetadata: []*iam.AccessKeyMetadata{}}, nil)
}
func mockListAccessKeys(mockAWSClient *mockaws.MockClient, accessKeyID string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(testAWSUser)}).Return(&iam.ListAccessKeysOutput{AccessKeyMetadata: []*iam.AccessKeyMetadata{{AccessKeyId: aws.String(accessKeyID)}}}, nil)
}
func mockCreateUser(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().CreateUser(&iam.CreateUserInput{UserName: aws.String(testAWSUser)}).Return(&iam.CreateUserOutput{User: &iam.User{UserName: aws.String(testAWSUser), UserId: aws.String(testAWSUserID), Arn: aws.String(testAWSARN)}}, nil)
}
func mockCreateAccessKey(mockAWSClient *mockaws.MockClient, accessKeyID, secretAccessKey string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().CreateAccessKey(&iam.CreateAccessKeyInput{UserName: aws.String(testAWSUser)}).Return(&iam.CreateAccessKeyOutput{AccessKey: &iam.AccessKey{AccessKeyId: aws.String(accessKeyID), SecretAccessKey: aws.String(secretAccessKey)}}, nil)
}
func mockTagUser(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().TagUser(&iam.TagUserInput{UserName: aws.String(testAWSUser), Tags: []*iam.Tag{{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", testInfraName)), Value: aws.String("owned")}}}).Return(&iam.TagUserOutput{}, nil)
}
func mockTagUserLegacy(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().TagUser(&iam.TagUserInput{UserName: aws.String(testAWSUser), Tags: []*iam.Tag{{Key: aws.String(openshiftClusterIDKey), Value: aws.String(testClusterID)}}}).Return(&iam.TagUserOutput{}, nil)
}
func mockDeleteAccessKey(mockAWSClient *mockaws.MockClient, accessKeyID string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().DeleteAccessKey(&iam.DeleteAccessKeyInput{UserName: aws.String(testAWSUser), AccessKeyId: aws.String(accessKeyID)}).Return(&iam.DeleteAccessKeyOutput{}, nil)
}
func mockPutUserPolicy(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().PutUserPolicy(gomock.Any()).Return(&iam.PutUserPolicyOutput{}, nil)
}
func mockGetUserPolicy(mockAWSClient *mockaws.MockClient, policyDoc string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	policyDoc = url.QueryEscape(policyDoc)
	mockAWSClient.EXPECT().GetUserPolicy(gomock.Any()).Return(&iam.GetUserPolicyOutput{PolicyDocument: aws.String(policyDoc)}, nil)
}
func mockGetUserPolicyMissing(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().GetUserPolicy(gomock.Any()).Return(nil, awserr.New(iam.ErrCodeNoSuchEntityException, "no such policy", nil))
}
func testClusterVersion() *configv1.ClusterVersion {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: "version"}, Spec: configv1.ClusterVersionSpec{ClusterID: testClusterID}}
}
func mockSimulatePrincipalPolicySuccess(mockAWSClient *mockaws.MockClient) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	mockAWSClient.EXPECT().SimulatePrincipalPolicy(gomock.Any()).Return(&iam.SimulatePolicyResponse{EvaluationResults: []*iam.EvaluationResult{{EvalDecision: aws.String("allowed")}}}, nil)
}
func testInfrastructure(infraName string) *configv1.Infrastructure {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &configv1.Infrastructure{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}, Status: configv1.InfrastructureStatus{Platform: configv1.AWSPlatformType, InfrastructureName: infraName}}
}
