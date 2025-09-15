package controllers

import (
	"context"
	"os"

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
		log.Info("Using specified AWS region", "region", c.Region)
		opts = append(opts, config.WithRegion(c.Region))
	} else {
		// Try to get region from environment variables
		region := getDefaultRegion()
		if region != "" {
			log.Info("Using AWS region from environment", "region", region)
			opts = append(opts, config.WithRegion(region))
		} else {
			log.Info("No AWS region specified, will attempt to use instance metadata")
		}
	}

	// Set custom endpoint if specified (useful for testing or non-standard endpoints)
	if c.EndpointURL != "" {
		log.Info("Using custom AWS endpoint URL", "endpoint", c.EndpointURL)
		// Use proper endpoint resolver option
		opts = append(opts, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == secretsmanager.ServiceID {
					return aws.Endpoint{
						URL:           c.EndpointURL,
						SigningRegion: region,
					}, nil
				}
				// Fallback to default endpoint resolver
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}),
		))
	}

	// Load configuration with all credential providers in the chain
	log.Info("Loading AWS configuration")
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Error(err, "Failed to load AWS config")
		return nil, err
	}

	// Log the effective region being used
	log.Info("AWS configuration loaded", "region", cfg.Region)

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

// getDefaultRegion tries to get an AWS region from environment variables
func getDefaultRegion() string {
	possibleEnvVars := []string{"AWS_REGION", "AWS_DEFAULT_REGION"}
	for _, envVar := range possibleEnvVars {
		if region := os.Getenv(envVar); region != "" {
			return region
		}
	}
	return ""
}
