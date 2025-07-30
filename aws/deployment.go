package aws

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/Owloops/updo/utils"
)

//go:embed bootstrap
var embeddedLambdaBinary []byte

const (
	functionName        = "updo-executor"
	roleName            = "updo-lambda-execution-role"
	iamRoleWaitTime     = 10 * time.Second
	lambdaTimeout       = 30
	lambdaMemoryMB      = 128
	awsOperationTimeout = 30 * time.Second
)

var defaultRegions = []string{
	"us-east-1",
	"us-west-2",
	"eu-central-1",
	"ap-southeast-1",
}

type Deployer struct {
	lambdaClient *lambda.Client
	iamClient    *iam.Client
	stsClient    *sts.Client
	region       string
	accountID    string
}

type DeploymentResult struct {
	Region      string `json:"region"`
	FunctionArn string `json:"function_arn"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

type DeploymentOptions struct {
	Profile    string
	Sequential bool
}

func NewDeployer(region string, profile string) (*Deployer, error) {
	configOpts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if profile != "" {
		configOpts = append(configOpts, config.WithSharedConfigProfile(profile))
	}
	ctx, cancel := context.WithTimeout(context.Background(), awsOperationTimeout)
	defer cancel()
	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for region %s: %w", region, err)
	}

	deployer := &Deployer{
		lambdaClient: lambda.NewFromConfig(cfg),
		iamClient:    iam.NewFromConfig(cfg),
		stsClient:    sts.NewFromConfig(cfg),
		region:       region,
	}

	accountID, err := deployer.getAccountID()
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS account ID: %w", err)
	}
	deployer.accountID = accountID

	return deployer, nil
}

func DeployToRegions(regions []string, options ...DeploymentOptions) []DeploymentResult {
	if len(regions) == 0 {
		regions = defaultRegions
	}

	var opts DeploymentOptions
	if len(options) > 0 {
		opts = options[0]
	}

	return executeRegionOperation(regions, opts, deployToRegion)
}

func deployToRegion(region, profile string) DeploymentResult {
	result := DeploymentResult{Region: region}

	deployer, err := NewDeployer(region, profile)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	functionArn, err := deployer.Deploy()
	if err != nil {
		result.Error = err.Error()
	} else {
		result.FunctionArn = functionArn
		result.Success = true
	}

	return result
}

func DestroyFromRegions(regions []string, options ...DeploymentOptions) []DeploymentResult {
	if len(regions) == 0 {
		regions = defaultRegions
	}

	var opts DeploymentOptions
	if len(options) > 0 {
		opts = options[0]
	}

	results := executeRegionOperation(regions, opts, destroyFromRegion)

	if shouldCleanupIAMRole(regions, results) {
		cleanupIAMRole(regions[0], opts.Profile)
	}

	return results
}

type regionOperationFunc func(region, profile string) DeploymentResult

func executeRegionOperation(regions []string, opts DeploymentOptions, operation regionOperationFunc) []DeploymentResult {
	results := make([]DeploymentResult, len(regions))

	if opts.Sequential {
		for i, region := range regions {
			results[i] = operation(region, opts.Profile)
		}
	} else {
		ch := make(chan DeploymentResult, len(regions))

		for _, region := range regions {
			go func(r string) {
				ch <- operation(r, opts.Profile)
			}(region)
		}

		for i := 0; i < len(regions); i++ {
			result := <-ch
			for j, region := range regions {
				if result.Region == region {
					results[j] = result
					break
				}
			}
		}
	}

	return results
}

func shouldCleanupIAMRole(regions []string, results []DeploymentResult) bool {
	if !IsDestroyingAllDefaultRegions(regions) {
		return false
	}

	for _, result := range results {
		if !result.Success {
			return false
		}
	}

	return len(regions) > 0
}

func IsDestroyingAllDefaultRegions(regions []string) bool {
	if len(regions) != len(defaultRegions) {
		return false
	}

	regionSet := make(map[string]bool)
	for _, region := range regions {
		regionSet[region] = true
	}

	for _, defaultRegion := range defaultRegions {
		if !regionSet[defaultRegion] {
			return false
		}
	}

	return true
}

func cleanupIAMRole(region, profile string) {
	deployer, err := NewDeployer(region, profile)
	if err == nil {
		err = deployer.destroyIAMRole()
		if err != nil {
			utils.Log.Warn(fmt.Sprintf("Failed to cleanup IAM role: %v", err))
		}
	}
}

func destroyFromRegion(region, profile string) DeploymentResult {
	result := DeploymentResult{Region: region}

	deployer, err := NewDeployer(region, profile)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	err = deployer.Destroy()
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	return result
}

func (d *Deployer) Deploy() (string, error) {
	utils.Log.Info(fmt.Sprintf("Deploying Lambda function to %s...", d.region))

	roleArn, err := d.ensureIAMRole()
	if err != nil {
		return "", fmt.Errorf("failed to ensure IAM role: %w", err)
	}

	functionArn, err := d.deployLambdaFunction(roleArn, embeddedLambdaBinary)
	if err != nil {
		return "", err
	}

	utils.Log.Success(fmt.Sprintf("Successfully deployed Lambda function: %s", functionArn))
	return functionArn, nil
}

func (d *Deployer) Destroy() error {
	funcName := fmt.Sprintf("%s-%s", functionName, d.region)
	ctx, cancel := context.WithTimeout(context.Background(), awsOperationTimeout)
	defer cancel()
	_, err := d.lambdaClient.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
		FunctionName: aws.String(funcName),
	})
	if err != nil {
		var notFoundErr *lambdatypes.ResourceNotFoundException
		if !strings.Contains(err.Error(), "Function not found") && !errors.As(err, &notFoundErr) {
			return fmt.Errorf("failed to delete Lambda function: %w", err)
		}
	}
	return nil
}

func (d *Deployer) destroyIAMRole() error {
	ctx, cancel := context.WithTimeout(context.Background(), awsOperationTimeout)
	defer cancel()
	if _, err := d.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
	}); err != nil {
		utils.Log.Warn(fmt.Sprintf("Failed to detach policy from role: %v", err))
	}

	_, err := d.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
		return fmt.Errorf("failed to delete IAM role: %w", err)
	}
	return nil
}

func (d *Deployer) getAccountID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), awsOperationTimeout)
	defer cancel()
	result, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}

func (d *Deployer) ensureIAMRole() (string, error) {
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", d.accountID, roleName)

	ctx, cancel := context.WithTimeout(context.Background(), awsOperationTimeout)
	defer cancel()
	if _, err := d.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}); err == nil {
		log.Printf("IAM role %s already exists", roleName)
		return roleArn, nil
	}

	log.Printf("Creating IAM role %s...", roleName)
	trustPolicyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}`
	_, err := d.iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicyDoc),
		Description:              aws.String("Execution role for updo Lambda function"),
		Tags: []iamtypes.Tag{
			{Key: aws.String("CreatedBy"), Value: aws.String("updo")},
			{Key: aws.String("Purpose"), Value: aws.String("updo-multi-region-monitoring")},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "EntityAlreadyExists") {
			log.Printf("IAM role %s already exists (created by another region)", roleName)
		} else {
			return "", fmt.Errorf("failed to create IAM role: %w", err)
		}
	}

	_, err = d.iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
	})
	if err != nil {
		return "", err
	}

	time.Sleep(iamRoleWaitTime)
	return roleArn, nil
}

func (d *Deployer) deployLambdaFunction(roleArn string, lambdaBinary []byte) (string, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	writer, err := zipWriter.Create("bootstrap")
	if err != nil {
		return "", fmt.Errorf("failed to create ZIP entry: %w", err)
	}
	if _, err := writer.Write(lambdaBinary); err != nil {
		return "", fmt.Errorf("failed to write Lambda binary: %w", err)
	}
	if err := zipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close ZIP: %w", err)
	}
	zipData := buf.Bytes()
	funcName := fmt.Sprintf("%s-%s", functionName, d.region)
	ctx, cancel := context.WithTimeout(context.Background(), awsOperationTimeout)
	defer cancel()

	if getOutput, err := d.lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(funcName),
	}); err == nil {
		log.Printf("Updating existing Lambda function %s...", funcName)
		_, err = d.lambdaClient.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(funcName),
			ZipFile:      zipData,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update Lambda function: %w", err)
		}
		return *getOutput.Configuration.FunctionArn, nil
	}

	log.Printf("Creating new Lambda function %s...", funcName)
	createOutput, err := d.lambdaClient.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(funcName),
		Runtime:      lambdatypes.RuntimeProvidedal2023,
		Role:         aws.String(roleArn),
		Handler:      aws.String("bootstrap"),
		Code: &lambdatypes.FunctionCode{
			ZipFile: zipData,
		},
		Architectures: []lambdatypes.Architecture{lambdatypes.ArchitectureArm64},
		MemorySize:    aws.Int32(lambdaMemoryMB),
		Timeout:       aws.Int32(lambdaTimeout),
		Description:   aws.String("updo website executor function"),
		Tags: map[string]string{
			"CreatedBy": "updo",
			"Purpose":   "updo-multi-region-monitoring",
			"Region":    d.region,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to deploy Lambda function: %w", err)
	}
	return *createOutput.FunctionArn, nil
}
