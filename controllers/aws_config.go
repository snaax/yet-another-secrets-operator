package controllers

import (
	"net/http"
	"time"

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
	// Create a custom HTTP client that allows non-localhost endpoints for EKS Pod Identity
	httpClient := &http.Client{
		Timeout: time.Second * 30,
	}

	// Create AWS config
	awsConfig := &aws.Config{
		HTTPClient: httpClient,
		MaxRetries: aws.Int(c.MaxRetries),
	}

	// Set region if specified
	if c.Region != "" {
		awsConfig.Region = aws.String(c.Region)
	}

	// Create session
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
	sess, err := session.NewSession()
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
