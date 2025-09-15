package config

import (
	"os"
	"strings"

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
	Region      string
	EndpointURL string
	MaxRetries  int
	Tags        map[string]string
}

// HealthConfig holds health server configuration
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
	// Initialize default tags
	defaultTags := make(map[string]string)
	defaultTags["managed-by"] = "yaso"

	return &OperatorConfig{
		AWS: AWSConfig{
			Region:      "",
			EndpointURL: "",
			MaxRetries:  5,
			Tags:        defaultTags,
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

	// Load tags from environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "AWS_TAG_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimPrefix(parts[0], "AWS_TAG_")
				c.AWS.Tags[strings.ToLower(key)] = parts[1]
			}
		}
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

// ToAWSConfig converts the config to a format usable by controllers
func (c *OperatorConfig) ToAWSConfig() AWSConfig {
	return AWSConfig{
		Region:      c.AWS.Region,
		EndpointURL: c.AWS.EndpointURL,
		MaxRetries:  c.AWS.MaxRetries,
		Tags:        c.AWS.Tags,
	}
}
