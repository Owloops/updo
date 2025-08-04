package aws

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
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

//go:embed bootstrap.zip
var embeddedLambdaZip []byte

const (
	_functionName         = "updo-executor"
	_roleName             = "updo-lambda-execution-role"
	_iamRoleWaitTime      = 10 * time.Second
	_lambdaTimeout        = 30
	_lambdaMemoryMB       = 128
	_awsOperationTimeout  = 30 * time.Second
	_functionCheckTimeout = 5 * time.Second
)

var _defaultRegions = []string{
	"us-east-1",      // N. Virginia
	"us-west-1",      // N. California
	"us-west-2",      // Oregon
	"eu-west-1",      // Ireland
	"eu-central-1",   // Frankfurt
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Sydney
	"ap-northeast-1", // Tokyo
	"ap-northeast-2", // Seoul
	"ap-south-1",     // Mumbai
	"sa-east-1",      // São Paulo
	"ca-central-1",   // Canada
	"eu-west-2",      // London
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
	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
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
		regions = _defaultRegions
	}

	validatedRegions := validateRegions(regions)

	var opts DeploymentOptions
	if len(options) > 0 {
		opts = options[0]
	}

	return executeRegionOperation(validatedRegions, opts, deployToRegion, "deploy")
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
		regions = _defaultRegions
	}

	var opts DeploymentOptions
	if len(options) > 0 {
		opts = options[0]
	}

	results := executeRegionOperation(regions, opts, destroyFromRegion, "destroy")

	if shouldCleanupIAMRole(regions, results) {
		cleanupIAMRole(regions[0], opts.Profile)
	}

	return results
}

type regionOperationFunc func(region, profile string) DeploymentResult

func executeRegionOperation(regions []string, opts DeploymentOptions, operation regionOperationFunc, operationType string) []DeploymentResult {
	results := make([]DeploymentResult, len(regions))
	total := len(regions)

	if opts.Sequential {
		operationName := "Deploying"
		progressLabel := "Deployment"
		if operationType == "destroy" {
			operationName = "Destroying"
			progressLabel = "Destroy"
		}
		utils.Log.Info(fmt.Sprintf("%s to %d regions sequentially...", operationName, total))
		for i, region := range regions {
			utils.Log.ProgressWithStatus(i, total, progressLabel, fmt.Sprintf("Processing %s", region))
			results[i] = operation(region, opts.Profile)
			utils.Log.ProgressWithStatus(i+1, total, progressLabel, fmt.Sprintf("Completed %s", region))
		}
	} else {
		ch := make(chan DeploymentResult, len(regions))
		completed := 0
		successful := 0

		spinnerStop := make(chan bool)
		operationName := "Deploying"
		progressLabel := "Deployment"
		if operationType == "destroy" {
			operationName = "Destroying"
			progressLabel = "Destroy"
		}
		go utils.Log.Spinner(fmt.Sprintf("%s to %d regions in parallel...", operationName, total), spinnerStop)

		for _, region := range regions {
			go func(r string) {
				ch <- operation(r, opts.Profile)
			}(region)
		}

		for range len(regions) {
			result := <-ch
			completed++
			if result.Success {
				successful++
			}

			if completed == 1 {
				spinnerStop <- true
				close(spinnerStop)
			}

			utils.Log.ProgressWithStatus(successful, total, progressLabel,
				fmt.Sprintf("%d/%d completed (%d successful)", completed, total, successful))

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
	if len(regions) != len(_defaultRegions) {
		return false
	}

	regionSet := make(map[string]bool, len(regions))
	for _, region := range regions {
		regionSet[region] = true
	}

	for _, defaultRegion := range _defaultRegions {
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
	roleArn, err := d.ensureIAMRole()
	if err != nil {
		return "", fmt.Errorf("failed to ensure IAM role: %w", err)
	}

	functionArn, err := d.deployLambdaFunction(roleArn, embeddedLambdaZip)
	if err != nil {
		return "", err
	}

	return functionArn, nil
}

func (d *Deployer) Destroy() error {
	funcName := fmt.Sprintf("%s-%s", _functionName, d.region)
	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
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
	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
	defer cancel()
	if _, err := d.iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
		RoleName:  aws.String(_roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
	}); err != nil {
		utils.Log.Warn(fmt.Sprintf("Failed to detach policy from role: %v", err))
	}

	_, err := d.iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: aws.String(_roleName),
	})
	if err != nil && !strings.Contains(err.Error(), "NoSuchEntity") {
		return fmt.Errorf("failed to delete IAM role: %w", err)
	}
	return nil
}

func (d *Deployer) getAccountID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
	defer cancel()
	result, err := d.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *result.Account, nil
}

func (d *Deployer) ensureIAMRole() (string, error) {
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", d.accountID, _roleName)

	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
	defer cancel()
	if _, err := d.iamClient.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(_roleName),
	}); err == nil {
		return roleArn, nil
	}
	trustPolicyDoc := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}`
	_, err := d.iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(_roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicyDoc),
		Description:              aws.String("Execution role for updo Lambda function"),
		Tags: []iamtypes.Tag{
			{Key: aws.String("CreatedBy"), Value: aws.String("updo")},
			{Key: aws.String("Purpose"), Value: aws.String("updo-multi-region-monitoring")},
		},
	})
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists") {
		return "", fmt.Errorf("failed to create IAM role: %w", err)
	}

	_, err = d.iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(_roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
	})
	if err != nil {
		return "", err
	}

	time.Sleep(_iamRoleWaitTime)
	return roleArn, nil
}

func (d *Deployer) deployLambdaFunction(roleArn string, lambdaZip []byte) (string, error) {
	zipData := lambdaZip
	funcName := fmt.Sprintf("%s-%s", _functionName, d.region)
	ctx, cancel := context.WithTimeout(context.Background(), _awsOperationTimeout)
	defer cancel()

	if getOutput, err := d.lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(funcName),
	}); err == nil {
		_, err = d.lambdaClient.UpdateFunctionCode(ctx, &lambda.UpdateFunctionCodeInput{
			FunctionName: aws.String(funcName),
			ZipFile:      zipData,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update Lambda function: %w", err)
		}
		return *getOutput.Configuration.FunctionArn, nil
	}
	createOutput, err := d.lambdaClient.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(funcName),
		Runtime:      lambdatypes.RuntimeProvidedal2023,
		Role:         aws.String(roleArn),
		Handler:      aws.String("bootstrap"),
		Code: &lambdatypes.FunctionCode{
			ZipFile: zipData,
		},
		Architectures: []lambdatypes.Architecture{lambdatypes.ArchitectureArm64},
		MemorySize:    aws.Int32(_lambdaMemoryMB),
		Timeout:       aws.Int32(_lambdaTimeout),
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

func GetDeployedRegions(profile string) ([]string, error) {
	var deployedRegions []string

	allRegions := _defaultRegions

	type regionResult struct {
		region string
		exists bool
	}

	resultChan := make(chan regionResult, len(allRegions))

	for _, region := range allRegions {
		go func(r string) {
			exists, _ := checkFunctionExists(r, profile)
			resultChan <- regionResult{region: r, exists: exists}
		}(region)
	}

	for range len(allRegions) {
		result := <-resultChan
		if result.exists {
			deployedRegions = append(deployedRegions, result.region)
		}
	}

	return deployedRegions, nil
}

func checkFunctionExists(region, profile string) (bool, error) {
	deployer, err := NewDeployer(region, profile)
	if err != nil {
		return false, err
	}

	funcName := fmt.Sprintf("%s-%s", _functionName, region)
	ctx, cancel := context.WithTimeout(context.Background(), _functionCheckTimeout)
	defer cancel()

	_, err = deployer.lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(funcName),
	})

	if err != nil {
		return false, nil
	}

	return true, nil
}

func validateRegions(regions []string) []string {
	var validRegions []string
	var unsupportedRegions []string

	defaultRegionSet := make(map[string]bool, len(_defaultRegions))
	for _, region := range _defaultRegions {
		defaultRegionSet[region] = true
	}

	for _, region := range regions {
		if defaultRegionSet[region] {
			validRegions = append(validRegions, region)
		} else {
			unsupportedRegions = append(unsupportedRegions, region)
		}
	}

	if len(unsupportedRegions) > 0 {
		utils.Log.Warn(fmt.Sprintf("Unsupported regions (not in our 13 locations): %s",
			strings.Join(unsupportedRegions, ", ")))
		utils.Log.Plain("Supported regions: us-east-1, us-west-1, us-west-2, eu-west-1, eu-central-1,")
		utils.Log.Plain("  ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-northeast-2,")
		utils.Log.Plain("  ap-south-1, sa-east-1, ca-central-1, eu-west-2")
		utils.Log.Plain("Continuing with supported regions only...")
	}

	return validRegions
}
