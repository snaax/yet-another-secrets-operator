package config

import (
	"os"

	"github.com/spf13/pflag"
)

// OperatorConfig holds all configuration for the operator
type OperatorConfig struct {
	AWS    AWSConfig
	Health HealthConfig
	Leader LeaderElectionConfig
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Region       string
	EndpointURL  string
	MaxRetries   int
	SkipConnTest bool
}

// HealthConfig holds health and metrics server configuration
type HealthConfig struct {
	ProbeBindAddress string
}

// LeaderElectionConfig holds leader election configuration
type LeaderElectionConfig struct {
	Enabled bool
	ID      string
}

// NewDefaultConfig returns a config with default values
func NewDefaultConfig() *OperatorConfig {
	return &OperatorConfig{
		AWS: AWSConfig{
			Region:       "",
			EndpointURL:  "",
			MaxRetries:   5,
			SkipConnTest: false,
		},
		Health: HealthConfig{
			ProbeBindAddress: ":8081",
		},
		Leader: LeaderElectionConfig{
			Enabled: false,
			ID:      "aso.yaso.io",
		},
	}
}

// AddFlags adds all config flags to the provided flag set
func (c *OperatorConfig) AddFlags(flags *pflag.FlagSet) {
	// AWS flags
	flags.StringVar(&c.AWS.Region, "aws-region", c.AWS.Region, "AWS Region to use")
	flags.StringVar(&c.AWS.EndpointURL, "aws-endpoint", c.AWS.EndpointURL, "Custom AWS endpoint URL")
	flags.IntVar(&c.AWS.MaxRetries, "aws-max-retries", c.AWS.MaxRetries, "Maximum number of AWS API retries")
	flags.BoolVar(&c.AWS.SkipConnTest, "skip-aws-test", c.AWS.SkipConnTest, "Skip testing AWS connectivity at startup")

	// Health flags
	flags.StringVar(&c.Health.ProbeBindAddress, "health-probe-bind-address", c.Health.ProbeBindAddress, "The address the probe endpoint binds to.")

	// Leader election flags
	flags.BoolVar(&c.Leader.Enabled, "leader-elect", c.Leader.Enabled, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
}

// LoadFromEnv loads config values from environment variables
func (c *OperatorConfig) LoadFromEnv() {
	// AWS Region
	if c.AWS.Region == "" {
		c.AWS.Region = GetDefaultRegion()
	}

	// AWS Endpoint URL
	if c.AWS.EndpointURL == "" {
		c.AWS.EndpointURL = os.Getenv("AWS_ENDPOINT_URL")
	}

	// Allow environment variable to override skip connection test
	if os.Getenv("SKIP_AWS_CONN_TEST") == "true" {
		c.AWS.SkipConnTest = true
	}
}

// getDefaultRegion tries to get an AWS region from environment variables
func GetDefaultRegion() string {
	possibleEnvVars := []string{"AWS_REGION", "AWS_DEFAULT_REGION"}
	for _, envVar := range possibleEnvVars {
		if region := os.Getenv(envVar); region != "" {
			return region
		}
	}
	return ""
}

// ToAWSControllerConfig converts the config to a format usable by controllers
func (c *OperatorConfig) ToAWSControllerConfig() AWSControllerConfig {
	return AWSControllerConfig{
		Region:      c.AWS.Region,
		EndpointURL: c.AWS.EndpointURL,
		MaxRetries:  c.AWS.MaxRetries,
	}
}

// AWSControllerConfig is the AWS configuration structure used by controllers
type AWSControllerConfig struct {
	Region      string
	EndpointURL string
	MaxRetries  int
}
