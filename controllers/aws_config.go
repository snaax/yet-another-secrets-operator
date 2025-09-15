package controllers

import (
	"context"
	"fmt"
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
	// Determine the region to use
	region := c.Region
	if region == "" {
		region = getDefaultRegion()
	}

	// If we still don't have a region, use a default one for fallback
	if region == "" {
		region = "us-east-1" // Use a default region as fallback
		log.Info("No AWS region specified, using default fallback", "region", region)
	} else {
		log.Info("Using AWS region", "region", region)
	}

	// Create basic config options
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithRetryMaxAttempts(c.MaxRetries),
	}

	// Load configuration with explicit region
	log.Info("Loading AWS configuration")
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Error(err, "Failed to load AWS config")
		return nil, err
	}

	// Create SecretsManager client options
	var clientOpts []func(*secretsmanager.Options)

	// Set custom endpoint if specified
	if c.EndpointURL != "" {
		log.Info("Using custom endpoint URL", "endpoint", c.EndpointURL)
		clientOpts = append(clientOpts, func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(c.EndpointURL)
		})
	}

	// Create client with the options
	smClient := secretsmanager.NewFromConfig(cfg, clientOpts...)

	// Log the configured region
	log.Info("AWS SecretsManager client created", "region", cfg.Region)

	return smClient, nil
}

// GetCredentialProviderInfo returns information about which credential provider was used
func (c *AWSConfig) GetCredentialProviderInfo(ctx context.Context, log logr.Logger) (string, error) {
	// Determine the region to use
	region := c.Region
	if region == "" {
		region = getDefaultRegion()
	}

	if region == "" {
		region = "us-east-1" // Default fallback
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
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

// TestConnection attempts to list secrets to verify connectivity
func (c *AWSConfig) TestConnection(ctx context.Context, log logr.Logger) error {
	// Determine the region to use
	region := c.Region
	if region == "" {
		region = getDefaultRegion()
	}

	// Create basic config
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	log.Info("Testing AWS connectivity", "region", region)
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Error(err, "Failed to load config for connectivity test")
		return err
	}

	// Create client
	client := secretsmanager.NewFromConfig(cfg)

	// Test with ListSecrets which is simpler than GetSecretValue
	log.Info("Attempting to list secrets to verify connectivity")
	resp, err := client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
		MaxResults: aws.Int32(1), // Only need one to verify connection
	})

	if err != nil {
		log.Error(err, "Failed connectivity test")
		return fmt.Errorf("AWS connectivity test failed: %w", err)
	}

	log.Info("AWS connectivity test succeeded", "secretCount", len(resp.SecretList))
	return nil
}
