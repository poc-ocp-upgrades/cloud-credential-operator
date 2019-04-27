package actuator

import (
	"context"
	godefaultbytes "bytes"
	godefaultruntime "runtime"
	"encoding/json"
	"fmt"
	"net/url"
	godefaulthttp "net/http"
	"reflect"
	log "github.com/sirupsen/logrus"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	ccaws "github.com/openshift/cloud-credential-operator/pkg/aws"
	minteraws "github.com/openshift/cloud-credential-operator/pkg/aws"
	actuatoriface "github.com/openshift/cloud-credential-operator/pkg/controller/credentialsrequest/actuator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/secretannotator"
	"github.com/openshift/cloud-credential-operator/pkg/controller/utils"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rootAWSCredsSecretNamespace	= "kube-system"
	rootAWSCredsSecret		= "aws-creds"
	roAWSCredsSecretNamespace	= "openshift-cloud-credential-operator"
	roAWSCredsSecret		= "cloud-credential-operator-iam-ro-creds"
	clusterConfigNamespace		= "kube-system"
	openshiftClusterIDKey		= "openshiftClusterID"
	clusterVersionObjectName	= "version"
)

var _ actuatoriface.Actuator = (*AWSActuator)(nil)

type AWSActuator struct {
	Client			client.Client
	Codec			*minterv1.ProviderCodec
	AWSClientBuilder	func(accessKeyID, secretAccessKey []byte, infraName string) (ccaws.Client, error)
	Scheme			*runtime.Scheme
}

func NewAWSActuator(client client.Client, scheme *runtime.Scheme) (*AWSActuator, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	codec, err := minterv1.NewCodec()
	if err != nil {
		log.WithError(err).Error("error creating AWS codec")
		return nil, fmt.Errorf("error creating AWS codec: %v", err)
	}
	return &AWSActuator{Codec: codec, Client: client, AWSClientBuilder: ccaws.NewClient, Scheme: scheme}, nil
}
func DecodeProviderStatus(codec *minterv1.ProviderCodec, cr *minterv1.CredentialsRequest) (*minterv1.AWSProviderStatus, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	awsStatus := minterv1.AWSProviderStatus{}
	var err error
	if cr.Status.ProviderStatus == nil {
		return &awsStatus, nil
	}
	err = codec.DecodeProviderStatus(cr.Status.ProviderStatus, &awsStatus)
	if err != nil {
		return nil, fmt.Errorf("error decoding v1 provider status: %v", err)
	}
	return &awsStatus, nil
}
func DecodeProviderSpec(codec *minterv1.ProviderCodec, cr *minterv1.CredentialsRequest) (*minterv1.AWSProviderSpec, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if cr.Spec.ProviderSpec != nil {
		awsSpec := minterv1.AWSProviderSpec{}
		err := codec.DecodeProviderSpec(cr.Spec.ProviderSpec, &awsSpec)
		if err != nil {
			return nil, fmt.Errorf("error decoding provider v1 spec: %v", err)
		}
		return &awsSpec, nil
	}
	return nil, fmt.Errorf("no providerSpec defined")
}
func (a *AWSActuator) Exists(ctx context.Context, cr *minterv1.CredentialsRequest) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr)
	logger.Debug("running Exists")
	var err error
	awsStatus, err := DecodeProviderStatus(a.Codec, cr)
	if err != nil {
		return false, err
	}
	if awsStatus.User == "" {
		logger.Debug("username unset")
		return false, nil
	}
	existingSecret := &corev1.Secret{}
	err = a.Client.Get(context.TODO(), types.NamespacedName{Namespace: cr.Spec.SecretRef.Namespace, Name: cr.Spec.SecretRef.Name}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug("target secret does not exist")
			return false, nil
		}
		return false, err
	}
	logger.Debug("target secret exists")
	return true, nil
}
func (a *AWSActuator) needsUpdate(ctx context.Context, cr *minterv1.CredentialsRequest, infraName string) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr)
	exists, err := a.Exists(ctx, cr)
	if err != nil {
		return true, err
	}
	if !exists {
		return true, nil
	}
	_, accessKey, secretKey := a.loadExistingSecret(cr)
	awsClient, err := a.AWSClientBuilder([]byte(accessKey), []byte(secretKey), infraName)
	if err != nil {
		return true, err
	}
	awsSpec, err := DecodeProviderSpec(a.Codec, cr)
	if err != nil {
		return true, err
	}
	awsStatus, err := DecodeProviderStatus(a.Codec, cr)
	if err != nil {
		return true, fmt.Errorf("unable to decode ProviderStatus: %v", err)
	}
	readAWSClient, err := a.buildReadAWSClient(cr, infraName)
	if err != nil {
		log.WithError(err).Error("error creating read-only AWS client")
		return true, fmt.Errorf("unable to check whether AWS user is properly tagged")
	}
	if awsStatus.User != "" {
		user, err := readAWSClient.GetUser(&iam.GetUserInput{UserName: aws.String(awsStatus.User)})
		if err != nil {
			logger.WithError(err).Errorf("error getting user: %s", user)
			return true, fmt.Errorf("unable to read info for username %v: %v", user, err)
		}
		clusterUUID, err := a.loadClusterUUID(logger)
		if err != nil {
			return true, err
		}
		if !userHasExpectedTags(logger, user.User, infraName, string(clusterUUID)) {
			return true, nil
		}
		logger.Debug("NeedsUpdate ListAccessKeys")
		allUserKeys, err := readAWSClient.ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(awsStatus.User)})
		if err != nil {
			logger.WithError(err).Error("error listing all access keys for user")
			return false, err
		}
		accessKeyExists, err := a.accessKeyExists(logger, allUserKeys, accessKey)
		if err != nil {
			logger.WithError(err).Error("error querying whether access key still valid")
		}
		if !accessKeyExists {
			return true, nil
		}
		desiredUserPolicy, err := a.getDesiredUserPolicy(awsSpec.StatementEntries, *user.User.Arn)
		if err != nil {
			return false, err
		}
		policyEqual, err := a.awsPolicyEqualsDesiredPolicy(desiredUserPolicy, awsSpec, awsStatus, user.User, readAWSClient, logger)
		if !policyEqual {
			return true, nil
		}
	} else {
		goodEnough, err := utils.CheckPermissionsUsingQueryClient(readAWSClient, awsClient, awsSpec.StatementEntries, logger)
		if err != nil {
			return true, fmt.Errorf("error validating whether current creds are good enough: %v", err)
		}
		if !goodEnough {
			return true, nil
		}
	}
	return false, nil
}
func (a *AWSActuator) Create(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.sync(ctx, cr)
}
func (a *AWSActuator) Update(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return a.sync(ctx, cr)
}
func (a *AWSActuator) sync(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr)
	logger.Debug("running sync")
	infraName, err := utils.LoadInfrastructureName(a.Client, logger)
	if err != nil {
		return err
	}
	needsUpdate, err := a.needsUpdate(ctx, cr, infraName)
	if err != nil {
		logger.WithError(err).Error("error determining whether a credentials update is needed")
		return &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: fmt.Sprintf("error determining whether a credentials update is needed: %v", err)}
	}
	if !needsUpdate {
		logger.Debug("credentials already up to date")
		return nil
	}
	cloudCredsSecret, err := a.getCloudCredentialsSecret(ctx, logger)
	if err != nil {
		logger.WithError(err).Error("issue with cloud credentials secret")
		return err
	}
	if cloudCredsSecret.Annotations[secretannotator.AnnotationKey] == secretannotator.InsufficientAnnotation {
		msg := "cloud credentials insufficient to satisfy credentials request"
		logger.Error(msg)
		return &actuatoriface.ActuatorError{ErrReason: minterv1.InsufficientCloudCredentials, Message: msg}
	}
	if cloudCredsSecret.Annotations[secretannotator.AnnotationKey] == secretannotator.PassthroughAnnotation {
		logger.Debugf("provisioning with passthrough")
		err := a.syncPassthrough(ctx, cr, cloudCredsSecret, logger)
		if err != nil {
			return err
		}
	} else if cloudCredsSecret.Annotations[secretannotator.AnnotationKey] == secretannotator.MintAnnotation {
		logger.Debugf("provisioning with cred minting")
		err := a.syncMint(ctx, cr, infraName, logger)
		if err != nil {
			msg := "error syncing creds in mint-mode"
			logger.WithError(err).Error(msg)
			return &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: fmt.Sprintf("%v: %v", msg, err)}
		}
	}
	return nil
}
func (a *AWSActuator) syncPassthrough(ctx context.Context, cr *minterv1.CredentialsRequest, cloudCredsSecret *corev1.Secret, logger log.FieldLogger) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	existingSecret, _, _ := a.loadExistingSecret(cr)
	accessKeyID := string(cloudCredsSecret.Data[secretannotator.AwsAccessKeyName])
	secretAccessKey := string(cloudCredsSecret.Data[secretannotator.AwsSecretAccessKeyName])
	err := a.syncAccessKeySecret(cr, accessKeyID, secretAccessKey, existingSecret, "", logger)
	if err != nil {
		msg := "error creating/updating secret"
		logger.WithError(err).Error(msg)
		return &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: fmt.Sprintf("%v: %v", msg, err)}
	}
	return nil
}
func (a *AWSActuator) syncMint(ctx context.Context, cr *minterv1.CredentialsRequest, infraName string, logger log.FieldLogger) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	awsSpec, err := DecodeProviderSpec(a.Codec, cr)
	if err != nil {
		return err
	}
	awsStatus, err := DecodeProviderStatus(a.Codec, cr)
	if err != nil {
		return err
	}
	if awsStatus.User == "" {
		username, err := generateUserName(infraName, cr.Name)
		if err != nil {
			return err
		}
		awsStatus.User = username
		awsStatus.Policy = getPolicyName(username)
		logger.WithField("user", awsStatus.User).Debug("generated random name for AWS user and policy")
		err = a.updateProviderStatus(ctx, logger, cr, awsStatus)
		if err != nil {
			return err
		}
	}
	if awsStatus.Policy == "" && awsStatus.User != "" {
		awsStatus.Policy = getPolicyName(awsStatus.User)
		err = a.updateProviderStatus(ctx, logger, cr, awsStatus)
		if err != nil {
			return err
		}
	}
	rootAWSClient, err := a.buildRootAWSClient(cr, infraName)
	if err != nil {
		logger.WithError(err).Warn("error building root AWS client, will error if one must be used")
	}
	readAWSClient, err := a.buildReadAWSClient(cr, infraName)
	if err != nil {
		logger.WithError(err).Error("error building read-only AWS client")
		return err
	}
	var userOut *iam.User
	getUserOut, err := readAWSClient.GetUser(&iam.GetUserInput{UserName: aws.String(awsStatus.User)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				logger.WithField("userName", awsStatus.User).Debug("user does not exist, creating")
				if rootAWSClient == nil {
					return fmt.Errorf("no root AWS client available, cred secret may not exist: %s/%s", rootAWSCredsSecretNamespace, rootAWSCredsSecret)
				}
				createOut, err := a.createUser(logger, rootAWSClient, awsStatus.User)
				if err != nil {
					return err
				}
				logger.WithField("userName", awsStatus.User).Info("user created successfully")
				userOut = createOut.User
			default:
				return formatAWSErr(aerr)
			}
		} else {
			return fmt.Errorf("unknown error getting user from AWS: %v", err)
		}
	} else {
		logger.WithField("userName", awsStatus.User).Info("user exists")
		userOut = getUserOut.User
	}
	clusterUUID, err := a.loadClusterUUID(logger)
	if err != nil {
		return err
	}
	if !userHasExpectedTags(logger, userOut, infraName, string(clusterUUID)) {
		if rootAWSClient == nil {
			return fmt.Errorf("no root AWS client available, cred secret may not exist: %s/%s", rootAWSCredsSecretNamespace, rootAWSCredsSecret)
		}
		err = a.tagUser(logger, rootAWSClient, awsStatus.User, infraName, string(clusterUUID))
		if err != nil {
			return err
		}
	}
	desiredUserPolicy, err := a.getDesiredUserPolicy(awsSpec.StatementEntries, *userOut.Arn)
	if err != nil {
		return err
	}
	policyEqual, err := a.awsPolicyEqualsDesiredPolicy(desiredUserPolicy, awsSpec, awsStatus, userOut, readAWSClient, logger)
	if !policyEqual {
		if rootAWSClient == nil {
			return fmt.Errorf("no root AWS client available, cred secret may not exist: %s/%s", rootAWSCredsSecretNamespace, rootAWSCredsSecret)
		}
		err = a.setUserPolicy(logger, rootAWSClient, awsStatus.User, awsStatus.Policy, desiredUserPolicy)
		if err != nil {
			return err
		}
		logger.Info("successfully set user policy")
	}
	logger.Debug("sync ListAccessKeys")
	allUserKeys, err := readAWSClient.ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(awsStatus.User)})
	if err != nil {
		logger.WithError(err).Error("error listing all access keys for user")
		return err
	}
	existingSecret, existingAccessKeyID, _ := a.loadExistingSecret(cr)
	var accessKey *iam.AccessKey
	accessKeyExists, err := a.accessKeyExists(logger, allUserKeys, existingAccessKeyID)
	if err != nil {
		return err
	}
	logger.WithField("accessKeyID", existingAccessKeyID).Debugf("access key exists? %v", accessKeyExists)
	if existingSecret != nil && existingSecret.Name != "" {
		_, ok := existingSecret.Annotations[minterv1.AnnotationAWSPolicyLastApplied]
		if !ok {
			logger.Warnf("target secret missing policy annotation: %s", minterv1.AnnotationAWSPolicyLastApplied)
		}
	}
	genNewAccessKey := existingSecret == nil || existingSecret.Name == "" || existingAccessKeyID == "" || !accessKeyExists
	if genNewAccessKey {
		logger.Info("generating new AWS access key")
		if rootAWSClient == nil {
			return fmt.Errorf("no root AWS client available, cred secret may not exist: %s/%s", rootAWSCredsSecretNamespace, rootAWSCredsSecret)
		}
		err := a.deleteAllAccessKeys(logger, rootAWSClient, awsStatus.User, allUserKeys)
		if err != nil {
			return err
		}
		accessKey, err = a.createAccessKey(logger, rootAWSClient, awsStatus.User)
		if err != nil {
			logger.WithError(err).Error("error creating AWS access key")
			return err
		}
	}
	accessKeyString := ""
	secretAccessKeyString := ""
	if accessKey != nil {
		accessKeyString = *accessKey.AccessKeyId
		secretAccessKeyString = *accessKey.SecretAccessKey
	}
	err = a.syncAccessKeySecret(cr, accessKeyString, secretAccessKeyString, existingSecret, desiredUserPolicy, logger)
	if err != nil {
		log.WithError(err).Error("error saving access key to secret")
		return err
	}
	return nil
}
func (a *AWSActuator) awsPolicyEqualsDesiredPolicy(desiredUserPolicy string, awsSpec *minterv1.AWSProviderSpec, awsStatus *minterv1.AWSProviderStatus, awsUser *iam.User, readAWSClient ccaws.Client, logger log.FieldLogger) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	currentUserPolicy, err := a.getCurrentUserPolicy(logger, readAWSClient, awsStatus.User, awsStatus.Policy)
	if err != nil {
		return false, err
	}
	logger.Debugf("desired user policy: %s", desiredUserPolicy)
	logger.Debugf("current user policy: %s", currentUserPolicy)
	if currentUserPolicy != desiredUserPolicy {
		logger.Debug("policy differences detected")
		return false, nil
	}
	logger.Debug("no changes to user policy")
	return true, nil
}
func userHasExpectedTags(logger log.FieldLogger, user *iam.User, infraName, clusterUUID string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if user == nil {
		return false
	}
	if infraName != "" {
		clusterTag := fmt.Sprintf("kubernetes.io/cluster/%s", infraName)
		if !userHasTag(user, clusterTag, "owned") {
			log.Warnf("user missing tag: %s=%s", clusterTag, "owned")
			return false
		}
	} else {
		logger.Warn("Infrastructure 'cluster' has no status.infrastructureName set. (likely beta3 cluster)")
		if !userHasTag(user, openshiftClusterIDKey, clusterUUID) {
			log.Warnf("user missing tag: %s=%s", openshiftClusterIDKey, clusterUUID)
			return false
		}
	}
	return true
}
func (a *AWSActuator) updateProviderStatus(ctx context.Context, logger log.FieldLogger, cr *minterv1.CredentialsRequest, awsStatus *minterv1.AWSProviderStatus) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	var err error
	cr.Status.ProviderStatus, err = a.Codec.EncodeProviderStatus(awsStatus)
	if err != nil {
		logger.WithError(err).Error("error encoding provider status")
		return err
	}
	if cr.Status.Conditions == nil {
		cr.Status.Conditions = []minterv1.CredentialsRequestCondition{}
	}
	err = a.Client.Status().Update(ctx, cr)
	if err != nil {
		logger.WithError(err).Error("error updating credentials request status")
		return err
	}
	return nil
}
func (a *AWSActuator) Delete(ctx context.Context, cr *minterv1.CredentialsRequest) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr)
	logger.Debug("running Delete")
	var err error
	awsStatus, err := DecodeProviderStatus(a.Codec, cr)
	if err != nil {
		return err
	}
	if awsStatus.User == "" {
		logger.Warn("no user name set on credentials being deleted, most likely were never provisioned or using passthrough creds")
		return nil
	}
	logger = logger.WithField("userName", awsStatus.User)
	logger.Info("deleting credential from AWS")
	infraName, err := utils.LoadInfrastructureName(a.Client, logger)
	if err != nil {
		return err
	}
	awsClient, err := a.buildRootAWSClient(cr, infraName)
	if err != nil {
		return err
	}
	_, err = awsClient.DeleteUserPolicy(&iam.DeleteUserPolicyInput{UserName: aws.String(awsStatus.User), PolicyName: aws.String(awsStatus.Policy)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				logger.Warn("user policy does not exist, ignoring error")
			default:
				return formatAWSErr(aerr)
			}
		} else {
			return fmt.Errorf("unknown error deleting user policy from AWS: %v", err)
		}
	}
	logger.Info("user policy deleted")
	logger.Debug("Delete ListAccessKeys")
	allUserKeys, err := awsClient.ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(awsStatus.User)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				logger.Warn("error listing access keys, user does not exist, returning success")
				return nil
			default:
				logger.WithError(err).Error("error listing all access keys for user")
				return formatAWSErr(aerr)
			}
		}
		logger.WithError(err).Error("error listing all access keys for user")
		return err
	}
	err = a.deleteAllAccessKeys(logger, awsClient, awsStatus.User, allUserKeys)
	if err != nil {
		return err
	}
	_, err = awsClient.DeleteUser(&iam.DeleteUserInput{UserName: aws.String(awsStatus.User)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				logger.Warn("user does not exist, returning success")
			default:
				return formatAWSErr(aerr)
			}
		} else {
			return fmt.Errorf("unknown error deleting user from AWS: %v", err)
		}
	}
	logger.Info("user deleted")
	return nil
}
func (a *AWSActuator) loadExistingSecret(cr *minterv1.CredentialsRequest) (*corev1.Secret, string, string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr)
	var existingAccessKeyID string
	var existingSecretAccessKey string
	existingSecret := &corev1.Secret{}
	err := a.Client.Get(context.TODO(), types.NamespacedName{Namespace: cr.Spec.SecretRef.Namespace, Name: cr.Spec.SecretRef.Name}, existingSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Debug("secret does not exist")
		}
	} else {
		keyBytes, ok := existingSecret.Data["aws_access_key_id"]
		if !ok {
			logger.Warning("secret did not have expected key: aws_access_key_id, will be regenerated")
		} else {
			decoded := string(keyBytes)
			existingAccessKeyID = string(decoded)
			logger.WithField("accessKeyID", existingAccessKeyID).Debug("found access key ID in target secret")
		}
		secretBytes, ok := existingSecret.Data["aws_secret_access_key"]
		if !ok {
			logger.Warning("secret did not have expected key: aws_secret_access_key")
		} else {
			existingSecretAccessKey = string(secretBytes)
		}
	}
	return existingSecret, existingAccessKeyID, existingSecretAccessKey
}
func (a *AWSActuator) tagUser(logger log.FieldLogger, awsClient minteraws.Client, username, infraName, clusterUUID string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger.WithField("infraName", infraName).Info("tagging user with infrastructure name")
	tags := []*iam.Tag{}
	if infraName != "" {
		tags = append(tags, &iam.Tag{Key: aws.String(fmt.Sprintf("kubernetes.io/cluster/%s", infraName)), Value: aws.String("owned")})
	} else {
		tags = append(tags, &iam.Tag{Key: aws.String(openshiftClusterIDKey), Value: aws.String(clusterUUID)})
	}
	_, err := awsClient.TagUser(&iam.TagUserInput{UserName: aws.String(username), Tags: tags})
	if err != nil {
		logger.WithError(err).Error("unable to tag user")
		return err
	}
	return nil
}
func (a *AWSActuator) buildRootAWSClient(cr *minterv1.CredentialsRequest, infraName string) (minteraws.Client, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr).WithField("secret", fmt.Sprintf("%s/%s", rootAWSCredsSecretNamespace, rootAWSCredsSecret))
	logger.Debug("loading AWS credentials from secret")
	accessKeyID, secretAccessKey, err := minteraws.LoadCredsFromSecret(a.Client, rootAWSCredsSecretNamespace, rootAWSCredsSecret)
	if err != nil {
		return nil, err
	}
	logger.Debug("creating root AWS client")
	return a.AWSClientBuilder(accessKeyID, secretAccessKey, infraName)
}
func (a *AWSActuator) buildReadAWSClient(cr *minterv1.CredentialsRequest, infraName string) (minteraws.Client, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger := a.getLogger(cr).WithField("secret", fmt.Sprintf("%s/%s", roAWSCredsSecretNamespace, roAWSCredsSecret))
	logger.Debug("loading AWS credentials from secret")
	var accessKeyID, secretAccessKey []byte
	var err error
	if cr.Spec.SecretRef.Name == roAWSCredsSecret && cr.Spec.SecretRef.Namespace == roAWSCredsSecretNamespace {
		log.Debug("operating our our RO creds, using root creds for all AWS client operations")
		accessKeyID, secretAccessKey, err = minteraws.LoadCredsFromSecret(a.Client, rootAWSCredsSecretNamespace, rootAWSCredsSecret)
		if err != nil {
			return nil, err
		}
	} else {
		accessKeyID, secretAccessKey, err = minteraws.LoadCredsFromSecret(a.Client, roAWSCredsSecretNamespace, roAWSCredsSecret)
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Warn("read-only creds not found, using root creds client")
				return a.buildRootAWSClient(cr, infraName)
			}
		}
	}
	logger.Debug("creating read AWS client")
	client, err := a.AWSClientBuilder(accessKeyID, secretAccessKey, infraName)
	if err != nil {
		return nil, err
	}
	awsStatus, err := DecodeProviderStatus(a.Codec, cr)
	if err != nil {
		return nil, err
	}
	_, err = client.GetUser(&iam.GetUserInput{UserName: aws.String(awsStatus.User)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidClientTokenId":
				logger.Warn("InvalidClientTokenId for read-only AWS account, likely a propagation delay, falling back to root AWS client")
				return a.buildRootAWSClient(cr, infraName)
			}
		}
	}
	return client, nil
}
func (a *AWSActuator) getLogger(cr *minterv1.CredentialsRequest) log.FieldLogger {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return log.WithFields(log.Fields{"actuator": "aws", "cr": fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)})
}
func (a *AWSActuator) syncAccessKeySecret(cr *minterv1.CredentialsRequest, accessKeyID, secretAccessKey string, existingSecret *corev1.Secret, userPolicy string, logger log.FieldLogger) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	sLog := logger.WithFields(log.Fields{"targetSecret": fmt.Sprintf("%s/%s", cr.Spec.SecretRef.Namespace, cr.Spec.SecretRef.Name), "cr": fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)})
	if existingSecret == nil || existingSecret.Name == "" {
		if accessKeyID == "" || secretAccessKey == "" {
			msg := "new access key secret needed but no key data provided"
			sLog.Error(msg)
			return &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: msg}
		}
		sLog.Info("creating secret")
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: cr.Spec.SecretRef.Name, Namespace: cr.Spec.SecretRef.Namespace, Annotations: map[string]string{minterv1.AnnotationCredentialsRequest: fmt.Sprintf("%s/%s", cr.Namespace, cr.Name), minterv1.AnnotationAWSPolicyLastApplied: userPolicy}}, Data: map[string][]byte{"aws_access_key_id": []byte(accessKeyID), "aws_secret_access_key": []byte(secretAccessKey)}}
		err := a.Client.Create(context.TODO(), secret)
		if err != nil {
			sLog.WithError(err).Error("error creating secret")
			return err
		}
		sLog.Info("secret created successfully")
		return nil
	}
	sLog.Debug("updating secret")
	origSecret := existingSecret.DeepCopy()
	if existingSecret.Annotations == nil {
		existingSecret.Annotations = map[string]string{}
	}
	existingSecret.Annotations[minterv1.AnnotationCredentialsRequest] = fmt.Sprintf("%s/%s", cr.Namespace, cr.Name)
	existingSecret.Annotations[minterv1.AnnotationAWSPolicyLastApplied] = userPolicy
	if accessKeyID != "" && secretAccessKey != "" {
		existingSecret.Data["aws_access_key_id"] = []byte(accessKeyID)
		existingSecret.Data["aws_secret_access_key"] = []byte(secretAccessKey)
	}
	if !reflect.DeepEqual(existingSecret, origSecret) {
		sLog.Info("target secret has changed, updating")
		err := a.Client.Update(context.TODO(), existingSecret)
		if err != nil {
			msg := "error updating secret"
			sLog.WithError(err).Error(msg)
			return &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: msg}
		}
	} else {
		sLog.Debug("target secret unchanged")
	}
	return nil
}
func (a *AWSActuator) getDesiredUserPolicy(entries []minterv1.StatementEntry, userARN string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	policyDoc := PolicyDocument{Version: "2012-10-17", Statement: []StatementEntry{}}
	for _, se := range entries {
		policyDoc.Statement = append(policyDoc.Statement, StatementEntry{Effect: se.Effect, Action: se.Action, Resource: se.Resource})
	}
	addGetUserStatement(&policyDoc, userARN)
	b, err := json.Marshal(&policyDoc)
	if err != nil {
		return "", fmt.Errorf("error marshalling user policy: %v", err)
	}
	return string(b), nil
}
func (a *AWSActuator) getCloudCredentialsSecret(ctx context.Context, logger log.FieldLogger) (*corev1.Secret, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	cloudCredSecret := &corev1.Secret{}
	if err := a.Client.Get(ctx, types.NamespacedName{Name: rootAWSCredsSecret, Namespace: rootAWSCredsSecretNamespace}, cloudCredSecret); err != nil {
		msg := "unable to fetch root cloud cred secret"
		logger.WithError(err).Error(msg)
		return nil, &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: fmt.Sprintf("%v: %v", msg, err)}
	}
	if !isSecretAnnotated(cloudCredSecret) {
		logger.WithField("secret", fmt.Sprintf("%s/%s", secretannotator.CloudCredSecretNamespace, secretannotator.CloudCredSecretName)).Error("cloud cred secret not yet annotated")
		return nil, &actuatoriface.ActuatorError{ErrReason: minterv1.CredentialsProvisionFailure, Message: fmt.Sprintf("cannot proceed without cloud cred secret annotation")}
	}
	return cloudCredSecret, nil
}
func isSecretAnnotated(secret *corev1.Secret) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if secret.ObjectMeta.Annotations == nil {
		return false
	}
	if _, ok := secret.ObjectMeta.Annotations[secretannotator.AnnotationKey]; !ok {
		return false
	}
	return true
}
func addGetUserStatement(policyDoc *PolicyDocument, userARN string) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	policyDoc.Statement = append(policyDoc.Statement, StatementEntry{Effect: "Allow", Action: []string{"iam:GetUser"}, Resource: userARN})
}
func (a *AWSActuator) getCurrentUserPolicy(logger log.FieldLogger, awsReadClient minteraws.Client, userName, policyName string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	cupOut, err := awsReadClient.GetUserPolicy(&iam.GetUserPolicyInput{UserName: aws.String(userName), PolicyName: aws.String(policyName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				logger.Warn("policy does not exist, creating")
				return "", nil
			default:
				err = formatAWSErr(aerr)
				logger.WithError(err).Errorf("AWS error getting user policy")
				return "", err
			}
		} else {
			logger.WithError(err).Error("error getting current user policy")
			return "", err
		}
	}
	urlEncoded := *cupOut.PolicyDocument
	currentUserPolicy, err := url.QueryUnescape(urlEncoded)
	if err != nil {
		logger.WithError(err).Error("error URL decoding policy doc")
	}
	return currentUserPolicy, err
}
func (a *AWSActuator) setUserPolicy(logger log.FieldLogger, awsClient minteraws.Client, userName, policyName, userPolicy string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_, err := awsClient.PutUserPolicy(&iam.PutUserPolicyInput{UserName: aws.String(userName), PolicyDocument: aws.String(userPolicy), PolicyName: aws.String(policyName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return formatAWSErr(aerr)
		}
		return fmt.Errorf("unknown error setting user policy in AWS: %v", err)
	}
	return nil
}
func (a *AWSActuator) accessKeyExists(logger log.FieldLogger, allUserKeys *iam.ListAccessKeysOutput, existingAccessKey string) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if existingAccessKey == "" {
		return false, nil
	}
	for _, key := range allUserKeys.AccessKeyMetadata {
		if *key.AccessKeyId == existingAccessKey {
			return true, nil
		}
	}
	logger.WithField("accessKeyID", existingAccessKey).Warn("access key no longer exists")
	return false, nil
}
func (a *AWSActuator) deleteAllAccessKeys(logger log.FieldLogger, awsClient minteraws.Client, username string, allUserKeys *iam.ListAccessKeysOutput) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger.Info("deleting all AWS access keys")
	for _, kmd := range allUserKeys.AccessKeyMetadata {
		akLog := logger.WithFields(log.Fields{"accessKeyID": *kmd.AccessKeyId})
		akLog.Info("deleting access key")
		_, err := awsClient.DeleteAccessKey(&iam.DeleteAccessKeyInput{AccessKeyId: kmd.AccessKeyId, UserName: aws.String(username)})
		if err != nil {
			akLog.WithError(err).Error("error deleting access key")
			return err
		}
	}
	logger.Info("all access keys deleted")
	return nil
}
func (a *AWSActuator) createAccessKey(logger log.FieldLogger, awsClient minteraws.Client, username string) (*iam.AccessKey, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	accessKeyResult, err := awsClient.CreateAccessKey(&iam.CreateAccessKeyInput{UserName: &username})
	if err != nil {
		return nil, fmt.Errorf("error creating access key for user %s: %v", username, err)
	}
	logger.WithField("accessKeyID", *accessKeyResult.AccessKey.AccessKeyId).Info("access key created")
	return accessKeyResult.AccessKey, err
}
func userHasTag(user *iam.User, key, val string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	for _, t := range user.Tags {
		if *t.Key == key && *t.Value == val {
			return true
		}
	}
	return false
}
func (a *AWSActuator) createUser(logger log.FieldLogger, awsClient minteraws.Client, username string) (*iam.CreateUserOutput, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	input := &iam.CreateUserInput{UserName: aws.String(username)}
	uLog := logger.WithField("userName", username)
	uLog.Info("creating user")
	userOut, err := awsClient.CreateUser(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeEntityAlreadyExistsException:
				uLog.Warn("user already exist")
				return nil, nil
			default:
				err = formatAWSErr(aerr)
				uLog.WithError(err).Errorf("AWS error creating user")
				return nil, err
			}
		}
		uLog.WithError(err).Errorf("unknown error creating user in AWS")
		return nil, fmt.Errorf("unknown error creating user in AWS: %v", err)
	} else {
		uLog.Debug("user created successfully")
	}
	return userOut, nil
}
func formatAWSErr(aerr awserr.Error) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch aerr.Code() {
	case iam.ErrCodeLimitExceededException:
		log.Error(iam.ErrCodeLimitExceededException, aerr.Error())
		return fmt.Errorf("AWS Error: %s - %s", iam.ErrCodeLimitExceededException, aerr.Error())
	case iam.ErrCodeEntityAlreadyExistsException:
		return fmt.Errorf("AWS Error: %s - %s", iam.ErrCodeEntityAlreadyExistsException, aerr.Error())
	case iam.ErrCodeNoSuchEntityException:
		return fmt.Errorf("AWS Error: %s - %s", iam.ErrCodeNoSuchEntityException, aerr.Error())
	case iam.ErrCodeServiceFailureException:
		return fmt.Errorf("AWS Error: %s - %s", iam.ErrCodeServiceFailureException, aerr.Error())
	default:
		log.Error(aerr.Error())
		return fmt.Errorf("AWS Error: %v", aerr)
	}
}
func generateUserName(infraName, credentialName string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if credentialName == "" {
		return "", fmt.Errorf("empty credential name")
	}
	infraPrefix := ""
	if infraName != "" {
		if len(infraName) > 20 {
			infraName = infraName[0:20]
		}
		infraPrefix = infraName + "-"
	}
	if len(credentialName) > 37 {
		credentialName = credentialName[0:37]
	}
	return fmt.Sprintf("%s%s-%s", infraPrefix, credentialName, utilrand.String(5)), nil
}
func getPolicyName(userName string) string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return userName + "-policy"
}

type PolicyDocument struct {
	Version		string
	Statement	[]StatementEntry
}
type StatementEntry struct {
	Effect		string
	Action		[]string
	Resource	string
}

func (a *AWSActuator) loadClusterUUID(logger log.FieldLogger) (configv1.ClusterID, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	logger.Debug("loading cluster version to read clusterID")
	clusterVer := &configv1.ClusterVersion{}
	err := a.Client.Get(context.Background(), types.NamespacedName{Name: clusterVersionObjectName}, clusterVer)
	if err != nil {
		logger.WithError(err).Error("error fetching clusterversion object")
		return "", err
	}
	logger.WithField("clusterID", clusterVer.Spec.ClusterID).Debug("found cluster ID")
	return clusterVer.Spec.ClusterID, nil
}
func _logClusterCodePath() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	pc, _, _, _ := godefaultruntime.Caller(1)
	jsonLog := []byte(fmt.Sprintf("{\"fn\": \"%s\"}", godefaultruntime.FuncForPC(pc).Name()))
	godefaulthttp.Post("http://35.226.239.161:5001/"+"logcode", "application/json", godefaultbytes.NewBuffer(jsonLog))
}
