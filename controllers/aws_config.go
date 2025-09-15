package controllers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
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
func (c *AWSConfig) CreateSecretsManagerClient(ctx context.Context, log logr.Logger) (*secretsmanager.Client, error) {
	// Create AWS SDK v2 config options
	opts := []func(*config.LoadOptions) error{
		config.WithRetryMaxAttempts(c.MaxRetries),
	}

	// Set region if specified
	if c.Region != "" {
		opts = append(opts, config.WithRegion(c.Region))
	}

	// Load configuration with all credential providers in the chain
	// This will automatically use:
	// 1. Environment variables
	// 2. Shared credentials file
	// 3. EKS Pod Identity or IAM Roles for Service Accounts
	// 4. EC2 Instance Profile
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Error(err, "Failed to load AWS config")
		return nil, err
	}

	// Set custom endpoint if specified (useful for testing or non-standard endpoints)
	if c.EndpointURL != "" {
		cfg.BaseEndpoint = aws.String(c.EndpointURL)
	}

	// Create and return the SecretsManager client
	return secretsmanager.NewFromConfig(cfg), nil
}

// GetCredentialProviderInfo returns information about which credential provider was used
func (c *AWSConfig) GetCredentialProviderInfo(ctx context.Context, log logr.Logger) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error(err, "Failed to load AWS config for credential check")
		return "", err
	}

	// Get credentials
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		log.Error(err, "Failed to get AWS credentials info")
		return "", err
	}

	return creds.Source, nil
}
