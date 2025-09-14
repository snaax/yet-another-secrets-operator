package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AGeneratorSpec defines the desired state of AGenerator
type AGeneratorSpec struct {
	// Length is the length of the generated value
	// +optional
	// +kubebuilder:default=16
	// +kubebuilder:validation:Minimum=1
	Length int `json:"length,omitempty"`

	// IncludeUppercase specifies if uppercase letters should be included
	// +optional
	// +kubebuilder:default=true
	IncludeUppercase bool `json:"includeUppercase,omitempty"`

	// IncludeLowercase specifies if lowercase letters should be included
	// +optional
	// +kubebuilder:default=true
	IncludeLowercase bool `json:"includeLowercase,omitempty"`

	// IncludeNumbers specifies if numbers should be included
	// +optional
	// +kubebuilder:default=true
	IncludeNumbers bool `json:"includeNumbers,omitempty"`

	// IncludeSpecialChars specifies if special characters should be included
	// +optional
	// +kubebuilder:default=true
	IncludeSpecialChars bool `json:"includeSpecialChars,omitempty"`

	// SpecialChars defines the set of special characters to use
	// +optional
	// +kubebuilder:default="!@#$%^&*()-_=+[]{}|;:,.<>?/"
	SpecialChars string `json:"specialChars,omitempty"`
}

// AGeneratorStatus defines the observed state of AGenerator
type AGeneratorStatus struct {
	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=agenerators,scope=Cluster,shortName=agen

// AGenerator is the Schema for the agenerators API
type AGenerator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AGeneratorSpec   `json:"spec,omitempty"`
	Status AGeneratorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AGeneratorList contains a list of AGenerator
type AGeneratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AGenerator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AGenerator{}, &AGeneratorList{})
}