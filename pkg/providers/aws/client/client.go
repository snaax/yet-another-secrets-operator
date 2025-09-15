package client

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-logr/logr"

	awsconfig "github.com/yaso/yet-another-secrets-operator/pkg/providers/aws/config"
)

// Client provides AWS operations
type AwsClient struct {
	Config awsconfig.AWSConfig
}

// NewClient creates a new AWS client
func NewClient(config awsconfig.AWSConfig) *AwsClient {
	return &AwsClient{
		Config: config,
	}
}

// CreateSecretsManagerClient creates a new AWS SecretsManager client
func (c *AwsClient) CreateSecretsManagerClient(ctx context.Context, log logr.Logger) (*secretsmanager.Client, error) {
	// Precedence: 1. Explicit config  2. Environment variables  3. Instance metadata
	region := c.determineRegion()
	endpoint := c.determineEndpoint()

	log.Info("Using AWS configuration", "region", region, "customEndpoint", endpoint != "")

	// Create basic config options
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithRetryMaxAttempts(c.Config.MaxRetries),
	}

	// Load configuration with explicit region
	log.V(1).Info("Loading AWS configuration")
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Error(err, "Failed to load AWS config")
		return nil, err
	}

	// Create SecretsManager client options
	var clientOpts []func(*secretsmanager.Options)

	// Set custom endpoint if specified
	if endpoint != "" {
		log.Info("Using custom endpoint URL", "endpoint", endpoint)
		clientOpts = append(clientOpts, func(o *secretsmanager.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	// Create client with the options
	smClient := secretsmanager.NewFromConfig(cfg, clientOpts...)

	// Log the configured region
	log.V(1).Info("AWS SecretsManager client created", "region", cfg.Region)

	return smClient, nil
}

// GetCredentialProviderInfo returns information about which credential provider was used
func (c *AwsClient) GetCredentialProviderInfo(ctx context.Context, log logr.Logger) (string, error) {
	// Determine the region to use
	region := c.determineRegion()

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

// Add helper methods for determining configuration
func (c *AwsClient) determineRegion() string {
	// Explicit config has highest priority
	if c.Config.Region != "" {
		return c.Config.Region
	}

	// Environment variables next
	return awsconfig.GetDefaultRegion()
}

func (c *AwsClient) determineEndpoint() string {
	// Explicit config has highest priority
	if c.Config.EndpointURL != "" {
		return c.Config.EndpointURL
	}

	// Environment variable next
	return os.Getenv("AWS_ENDPOINT_URL")
}

// TestConnection attempts to list secrets to verify connectivity
func (c *AwsClient) TestConnection(ctx context.Context, log logr.Logger) error {
	// Determine the region to use
	region := c.determineRegion()

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
