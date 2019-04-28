package utils

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	minterv1 "github.com/openshift/cloud-credential-operator/pkg/apis/cloudcredential/v1"
	ccaws "github.com/openshift/cloud-credential-operator/pkg/aws"
	"sigs.k8s.io/controller-runtime/pkg/client"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/iam"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
)

var (
	credMintingActions	= []string{"iam:CreateAccessKey", "iam:CreateUser", "iam:DeleteAccessKey", "iam:DeleteUser", "iam:DeleteUserPolicy", "iam:GetUser", "iam:GetUserPolicy", "iam:ListAccessKeys", "iam:PutUserPolicy", "iam:TagUser", "iam:SimulatePrincipalPolicy"}
	credPassthroughActions	= []string{"iam:GetUser", "iam:SimulatePrincipalPolicy", "elasticloadbalancing:DescribeLoadBalancers", "route53:ListHostedZones", "route53:ChangeResourceRecordSets", "tag:GetResources", "s3:CreateBucket", "s3:DeleteBucket", "s3:PutBucketTagging", "s3:GetBucketTagging", "s3:PutEncryptionConfiguration", "s3:GetEncryptionConfiguration", "s3:PutLifecycleConfiguration", "s3:GetLifecycleConfiguration", "s3:GetBucketLocation", "s3:ListBucket", "s3:HeadBucket", "s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:ListBucketMultipartUploads", "s3:AbortMultipartUpload", "ec2:DescribeImages", "ec2:DescribeVpcs", "ec2:DescribeSubnets", "ec2:DescribeAvailabilityZones", "ec2:DescribeSecurityGroups", "ec2:RunInstances", "ec2:DescribeInstances", "ec2:TerminateInstances", "elasticloadbalancing:RegisterInstancesWithLoadBalancer", "elasticloadbalancing:DescribeLoadBalancers", "elasticloadbalancing:DescribeTargetGroups", "elasticloadbalancing:RegisterTargets", "ec2:DescribeVpcs", "ec2:DescribeSubnets", "ec2:DescribeAvailabilityZones", "ec2:DescribeSecurityGroups", "ec2:RunInstances", "ec2:DescribeInstances", "ec2:TerminateInstances", "elasticloadbalancing:RegisterInstancesWithLoadBalancer", "elasticloadbalancing:DescribeLoadBalancers", "elasticloadbalancing:DescribeTargetGroups", "elasticloadbalancing:RegisterTargets", "iam:GetUser", "iam:GetUserPolicy", "iam:ListAccessKeys"}
	credentailRequestScheme	= runtime.NewScheme()
	credentialRequestCodec	= serializer.NewCodecFactory(credentailRequestScheme)
)

const (
	infrastructureConfigName = "cluster"
)

func init() {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if err := minterv1.AddToScheme(credentailRequestScheme); err != nil {
		panic(err)
	}
}
func LoadInfrastructureName(c client.Client, logger log.FieldLogger) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	infra := &configv1.Infrastructure{}
	err := c.Get(context.Background(), types.NamespacedName{Name: "cluster"}, infra)
	if err != nil {
		logger.WithError(err).Error("error loading Infrastructure config 'cluster'")
		return "", err
	}
	logger.Debugf("Loaded infrastructure name: %s", infra.Status.InfrastructureName)
	return infra.Status.InfrastructureName, nil
}
func CheckCloudCredCreation(awsClient ccaws.Client, logger log.FieldLogger) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return CheckPermissionsAgainstActions(awsClient, credMintingActions, logger)
}
func getClientDetails(awsClient ccaws.Client) (*iam.User, bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	rootUser := false
	user, err := awsClient.GetUser(nil)
	if err != nil {
		return nil, rootUser, fmt.Errorf("error querying username: %v", err)
	}
	parsed, err := arn.Parse(*user.User.Arn)
	if err != nil {
		return nil, rootUser, fmt.Errorf("error parsing user's ARN: %v", err)
	}
	if parsed.AccountID == *user.User.UserId {
		rootUser = true
	}
	return user.User, rootUser, nil
}
func CheckPermissionsUsingQueryClient(queryClient, targetClient ccaws.Client, statementEntries []minterv1.StatementEntry, logger log.FieldLogger) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	targetUser, isRoot, err := getClientDetails(targetClient)
	if err != nil {
		return false, fmt.Errorf("error gathering AWS credentials details: %v", err)
	}
	if isRoot {
		logger.Warn("Using the AWS account root user is not recommended: https://docs.aws.amazon.com/general/latest/gr/managing-aws-access-keys.html")
		return true, nil
	}
	allowList := []*string{}
	for _, statement := range statementEntries {
		for _, action := range statement.Action {
			allowList = append(allowList, aws.String(action))
		}
	}
	results, err := queryClient.SimulatePrincipalPolicy(&iam.SimulatePrincipalPolicyInput{PolicySourceArn: targetUser.Arn, ActionNames: allowList})
	if err != nil {
		return false, fmt.Errorf("error simulating policy: %v", err)
	}
	allClear := true
	for _, result := range results.EvaluationResults {
		if *result.EvalDecision != "allowed" {
			logger.WithField("action", *result.EvalActionName).Warning("Action not allowed with tested creds")
			allClear = false
		}
	}
	if !allClear {
		logger.Warningf("Tested creds not able to perform all requested actions")
		return false, nil
	}
	return true, nil
}
func CheckPermissionsAgainstStatementList(awsClient ccaws.Client, statementEntries []minterv1.StatementEntry, logger log.FieldLogger) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return CheckPermissionsUsingQueryClient(awsClient, awsClient, statementEntries, logger)
}
func CheckPermissionsAgainstActions(awsClient ccaws.Client, actionList []string, logger log.FieldLogger) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	statementList := []minterv1.StatementEntry{{Action: actionList, Resource: "*", Effect: "Allow"}}
	return CheckPermissionsAgainstStatementList(awsClient, statementList, logger)
}
func CheckCloudCredPassthrough(awsClient ccaws.Client, logger log.FieldLogger) (bool, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return CheckPermissionsAgainstActions(awsClient, credPassthroughActions, logger)
}
func readCredentialRequest(cr []byte) (*minterv1.CredentialsRequest, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	newObj, err := runtime.Decode(credentialRequestCodec.UniversalDecoder(minterv1.SchemeGroupVersion), cr)
	if err != nil {
		return nil, fmt.Errorf("error decoding credentialrequest: %v", err)
	}
	return newObj.(*minterv1.CredentialsRequest), nil
}
func getCredentialRequestStatements(crBytes []byte) ([]minterv1.StatementEntry, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	statementList := []minterv1.StatementEntry{}
	awsCodec, err := minterv1.NewCodec()
	if err != nil {
		return statementList, fmt.Errorf("error creating credentialrequest codec: %v", err)
	}
	cr, err := readCredentialRequest(crBytes)
	if err != nil {
		return statementList, err
	}
	awsSpec := minterv1.AWSProviderSpec{}
	err = awsCodec.DecodeProviderSpec(cr.Spec.ProviderSpec, &awsSpec)
	if err != nil {
		return statementList, fmt.Errorf("error decoding spec.ProviderSpec: %v", err)
	}
	statementList = append(statementList, awsSpec.StatementEntries...)
	return statementList, nil
}
