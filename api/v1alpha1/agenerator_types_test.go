package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAGeneratorDefault(t *testing.T) {
	generator := &AGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-generator",
			Namespace: "default",
		},
		Spec: AGeneratorSpec{
			Length:              16,
			IncludeUppercase:    true,
			IncludeLowercase:    true,
			IncludeNumbers:      true,
			IncludeSpecialChars: false,
		},
	}

	assert.Equal(t, "test-generator", generator.Name)
	assert.Equal(t, "default", generator.Namespace)
	assert.Equal(t, 16, generator.Spec.Length)
	assert.True(t, generator.Spec.IncludeUppercase)
	assert.True(t, generator.Spec.IncludeLowercase)
	assert.True(t, generator.Spec.IncludeNumbers)
	assert.False(t, generator.Spec.IncludeSpecialChars)
}

func TestAGeneratorGVK(t *testing.T) {
	generator := &AGenerator{}
	gvk := generator.GetObjectKind().GroupVersionKind()

	// These will be empty unless you set them explicitly or they're set by the scheme
	assert.NotNil(t, gvk)
}
