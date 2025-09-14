package controllers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-logr/logr"
)

// AWSConfigV2 holds AWS configuration for the operator using SDK v2
type AWSConfigV2 struct {
	Region      string
	EndpointURL string
	MaxRetries  int
}

// NewAWSConfigV2 creates a new AWS configuration with default values
func NewAWSConfigV2() *AWSConfigV2 {
	return &AWSConfigV2{
		Region:     "", // Empty means use the default from environment or instance metadata
		MaxRetries: 5,
	}
}

// CreateSecretsManagerClient creates a new AWS SecretsManager client using SDK v2
func (c *AWSConfigV2) CreateSecretsManagerClient(ctx context.Context, log logr.Logger) (*secretsmanager.Client, error) {
	// Create options for AWS config
	optFns := []func(*config.LoadOptions) error{
		config.WithRetryMaxAttempts(c.MaxRetries),
	}

	// Set region if specified
	if c.Region != "" {
		optFns = append(optFns, config.WithRegion(c.Region))
	}

	// Set custom endpoint URL if specified
	if c.EndpointURL != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == secretsmanager.ServiceID {
				return aws.Endpoint{
					URL:           c.EndpointURL,
					SigningRegion: region,
				}, nil
			}
			// Fallback to default resolver
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})
		optFns = append(optFns, config.WithEndpointResolverWithOptions(customResolver))
	}

	// Load the configuration with options
	log.Info("Creating AWS SDK v2 config")
	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		log.Error(err, "Failed to load AWS SDK v2 configuration")
		return nil, err
	}

	// Log information about which provider was used (for debugging)
	log.Info("AWS SDK v2 configuration created successfully")

	// Return a new SecretsManager client
	return secretsmanager.NewFromConfig(cfg), nil
}

// GetCredentialProviderInfo returns information about which credential provider was used (if available)
func (c *AWSConfigV2) GetCredentialProviderInfo(ctx context.Context, log logr.Logger) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error(err, "Failed to load AWS configuration for credential check")
		return "", err
	}

	// In SDK v2, getting detailed provider info requires checking the credentials directly
	_, err = cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", err
	}

	// We don't get the same level of provider detail in SDK v2
	return "AWS SDK v2 Credentials", nil
}
