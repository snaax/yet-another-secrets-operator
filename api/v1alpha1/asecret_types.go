package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ASecretSpec defines the desired state of ASecret
type ASecretSpec struct {
	// TargetSecretName is the name of the Kubernetes Secret to be created/managed
	TargetSecretName string `json:"targetSecretName"`

	// AwsSecretPath is the path in AWS SecretsManager where the secret is stored
	AwsSecretPath string `json:"awsSecretPath"`

	// KmsKeyId is the AWS KMS key ID or ARN to use for encrypting the secret in AWS Secrets Manager
	// If not specified, uses the default AWS managed key
	// +optional
	KmsKeyId string `json:"kmsKeyId,omitempty"`

	// Data contains the secret data. Each key must be a valid DNS subdomain name.
	// Values can be hardcoded or generated using a generator reference
	// +optional
	Data map[string]DataSource `json:"data,omitempty"`

	// Tags to apply to the AWS Secret (optional)
	// +optional
	Tags map[string]string `json:"tags,omitempty"`

	// OnlyImportRemote imports all values from remote provider only, do not create if missing
	// +optional
	OnlyImportRemote *bool `json:"onlyImportRemote,omitempty"`

	// ValueType specifies how the secret should be stored in AWS SecretsManager.
	// Allowed values: "kv" or "json". Default is "kv".
	// +kubebuilder:validation:Enum=kv;json
	// +optional
	ValueType string `json:"valueType,omitempty"`

	// RefreshInterval specifies how long the operator waits between each refresh/reconcile of this secret.
	// Default is "1h"
	// Example: "10m", "1h"
	// +optional
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`
}

// DataSource defines the source of the secret data
type DataSource struct {
	// Value is the hardcoded value for this key
	// +optional
	Value string `json:"value,omitempty"`

	// GeneratorRef refers to a AGenerator to generate values
	// +optional
	GeneratorRef *GeneratorReference `json:"generatorRef,omitempty"`

	// OnlyImportRemote imports value from remote provider only, do not create if missing
	// +optional
	OnlyImportRemote *bool `json:"onlyImportRemote,omitempty"`
}

// GeneratorReference contains the reference to a generator
type GeneratorReference struct {
	// Name of the generator
	Name string `json:"name"`
}

// ASecretStatus defines the observed state of ASecret
type ASecretStatus struct {
	// Conditions represent the latest available observations of the secret's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is the last time the secret was synced with AWS
	LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=asecrets,scope=Namespaced

// ASecret is the Schema for the asecrets API
type ASecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ASecretSpec   `json:"spec,omitempty"`
	Status ASecretStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ASecretList contains a list of ASecret
type ASecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ASecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ASecret{}, &ASecretList{})
}
