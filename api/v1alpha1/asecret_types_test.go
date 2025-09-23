package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestASecretDefault(t *testing.T) {
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "my-secret",
			AwsSecretPath:    "dev/myapp/secrets",
			Data: map[string]DataSource{ // Changed from ASecretData to DataSource
				"username": {
					Value: "admin",
				},
				"password": {
					GeneratorRef: &GeneratorReference{ // Changed from GeneratorRef to GeneratorReference
						Name: "password-generator",
					},
				},
			},
		},
	}

	assert.Equal(t, "test-secret", secret.Name)
	assert.Equal(t, "default", secret.Namespace)
	assert.Equal(t, "my-secret", secret.Spec.TargetSecretName)
	assert.Equal(t, "dev/myapp/secrets", secret.Spec.AwsSecretPath)
	assert.Len(t, secret.Spec.Data, 2)
	assert.Equal(t, "admin", secret.Spec.Data["username"].Value)
	assert.Equal(t, "password-generator", secret.Spec.Data["password"].GeneratorRef.Name)
}

func TestDataSourceWithValue(t *testing.T) {
	dataSource := DataSource{
		Value: "test-value",
	}

	assert.Equal(t, "test-value", dataSource.Value)
	assert.Nil(t, dataSource.GeneratorRef)
}

func TestDataSourceWithGeneratorRef(t *testing.T) {
	dataSource := DataSource{
		GeneratorRef: &GeneratorReference{
			Name: "my-generator",
		},
	}

	assert.Empty(t, dataSource.Value)
	assert.NotNil(t, dataSource.GeneratorRef)
	assert.Equal(t, "my-generator", dataSource.GeneratorRef.Name)
}

func TestASecretWithTags(t *testing.T) {
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret-with-tags",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "my-secret",
			AwsSecretPath:    "dev/myapp/secrets",
			KmsKeyId:         "alias/my-key",
			Tags: map[string]string{
				"environment": "dev",
				"team":        "platform",
			},
			Data: map[string]DataSource{
				"api-key": {
					Value: "secret-key",
				},
			},
		},
	}

	assert.Equal(t, "alias/my-key", secret.Spec.KmsKeyId)
	assert.Len(t, secret.Spec.Tags, 2)
	assert.Equal(t, "dev", secret.Spec.Tags["environment"])
	assert.Equal(t, "platform", secret.Spec.Tags["team"])
}

func TestASecretWithOnlyImportRemote(t *testing.T) {
	onlyImport := true
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret-import-only",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "my-secret",
			AwsSecretPath:    "dev/myapp/secrets",
			OnlyImportRemote: &onlyImport,
			Data: map[string]DataSource{
				"existing-key": {
					OnlyImportRemote: &onlyImport,
				},
			},
		},
	}

	assert.NotNil(t, secret.Spec.OnlyImportRemote)
	assert.True(t, *secret.Spec.OnlyImportRemote)
	assert.NotNil(t, secret.Spec.Data["existing-key"].OnlyImportRemote)
	assert.True(t, *secret.Spec.Data["existing-key"].OnlyImportRemote)
}
