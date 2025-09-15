package controllers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/go-logr/logr"
)

// AWSConfig holds AWS configuration for the operator
type AWSConfig struct {
	Region      string
	EndpointURL string
	MaxRetries  int
}

// NewAWSConfig creates a new AWS configuration with default values
func NewAWSConfig() *AWSConfig {
	return &AWSConfig{
		Region:     "", // Empty means use the default from environment or instance metadata
		MaxRetries: 5,
	}
}

// CreateSecretsManagerClient creates a new AWS SecretsManager client
func (c *AWSConfig) CreateSecretsManagerClient(log logr.Logger) (*secretsmanager.SecretsManager, error) {
	// Create AWS config with optional settings
	awsConfig := aws.NewConfig()

	// Set region if specified
	if c.Region != "" {
		awsConfig.WithRegion(c.Region)
	}

	// Set custom endpoint if specified (useful for testing or non-standard endpoints)
	if c.EndpointURL != "" {
		awsConfig.WithEndpoint(c.EndpointURL)
	}

	// Set max retries
	awsConfig.WithMaxRetries(c.MaxRetries)

	// Enable EC2 metadata service by setting EC2MetadataDisableTimeoutOverride to true
	// This allows the SDK to connect to the EC2 metadata service
	awsConfig.WithEC2MetadataDisableTimeoutOverride(true)

	// Create session using the default credential provider chain
	// This will automatically use:
	// 1. Environment variables
	// 2. Shared credentials file
	// 3. EKS Pod Identity or IAM Roles for Service Accounts
	// 4. EC2 Instance Profile
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		log.Error(err, "Failed to create AWS session")
		return nil, err
	}

	// Create and return the SecretsManager client
	return secretsmanager.New(sess), nil
}

// GetCredentialProviderInfo returns information about which credential provider was used
func (c *AWSConfig) GetCredentialProviderInfo(log logr.Logger) (string, error) {
	// Create AWS config with the EC2 metadata timeout override disabled
	awsConfig := aws.NewConfig().WithEC2MetadataDisableTimeoutOverride(true)

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		log.Error(err, "Failed to create AWS session for credential check")
		return "", err
	}

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		log.Error(err, "Failed to get AWS credentials info")
		return "", err
	}

	return creds.ProviderName, nil
}
