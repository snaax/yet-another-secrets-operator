package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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
			Data: map[string]DataSource{
				"username": {
					Value: "admin",
				},
				"password": {
					GeneratorRef: &GeneratorReference{
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

func TestASecretValueTypeKV(t *testing.T) {
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-value-type-kv",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "my-secret",
			AwsSecretPath:    "dev/myapp/secrets",
			ValueType:        "kv",
			Data: map[string]DataSource{
				"username": {Value: "admin"},
			},
		},
	}

	assert.Equal(t, "kv", secret.Spec.ValueType)
	assert.Contains(t, secret.Spec.Data, "username")
}

func TestASecretValueTypeJson(t *testing.T) {
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-value-type-json",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "my-secret",
			AwsSecretPath:    "dev/myapp/jsonsecret",
			ValueType:        "json",
			Data: map[string]DataSource{
				"json": {Value: `{"key":"value","another":"item"}`},
			},
		},
	}

	assert.Equal(t, "json", secret.Spec.ValueType)
	assert.Equal(t, `{"key":"value","another":"item"}`, secret.Spec.Data["json"].Value)
}

func TestASecretWithRefreshInterval(t *testing.T) {
	tenMin := metav1.Duration{Duration: 10 * time.Minute}
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-refresh-secret",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "refresh-secret",
			AwsSecretPath:    "dev/myapp/refresh",
			RefreshInterval:  &tenMin,
		},
	}

	assert.NotNil(t, secret.Spec.RefreshInterval)
	assert.Equal(t, 10*time.Minute, secret.Spec.RefreshInterval.Duration)
}

func TestASecretWithTargetSecretTemplate(t *testing.T) {
	secretType := corev1.SecretTypeTLS
	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret-template",
			Namespace: "default",
		},
		Spec: ASecretSpec{
			TargetSecretName: "my-tls-secret",
			AwsSecretPath:    "dev/myapp/tls",
			TargetSecretTemplate: &TargetSecretTemplate{
				Labels: map[string]string{
					"app":         "my-application",
					"environment": "production",
					"managed-by":  "yet-another-secrets-operator",
				},
				Annotations: map[string]string{
					"description":     "TLS certificate for my application",
					"owner":           "platform-team",
					"cert-manager.io": "true",
				},
				Type: &secretType,
			},
			Data: map[string]DataSource{
				"tls.crt": {Value: "cert-content"},
				"tls.key": {Value: "key-content"},
			},
		},
	}

	assert.NotNil(t, secret.Spec.TargetSecretTemplate)
	assert.Len(t, secret.Spec.TargetSecretTemplate.Labels, 3)
	assert.Equal(t, "my-application", secret.Spec.TargetSecretTemplate.Labels["app"])
	assert.Equal(t, "production", secret.Spec.TargetSecretTemplate.Labels["environment"])
	assert.Equal(t, "yet-another-secrets-operator", secret.Spec.TargetSecretTemplate.Labels["managed-by"])

	assert.Len(t, secret.Spec.TargetSecretTemplate.Annotations, 3)
	assert.Equal(t, "TLS certificate for my application", secret.Spec.TargetSecretTemplate.Annotations["description"])
	assert.Equal(t, "platform-team", secret.Spec.TargetSecretTemplate.Annotations["owner"])
	assert.Equal(t, "true", secret.Spec.TargetSecretTemplate.Annotations["cert-manager.io"])

	assert.NotNil(t, secret.Spec.TargetSecretTemplate.Type)
	assert.Equal(t, corev1.SecretTypeTLS, *secret.Spec.TargetSecretTemplate.Type)
}

func TestTargetSecretTemplateEmpty(t *testing.T) {
	template := &TargetSecretTemplate{}

	assert.Nil(t, template.Labels)
	assert.Nil(t, template.Annotations)
	assert.Nil(t, template.Type)
}

func TestTargetSecretTemplateLabelsOnly(t *testing.T) {
	template := &TargetSecretTemplate{
		Labels: map[string]string{
			"app": "test-app",
		},
	}

	assert.Len(t, template.Labels, 1)
	assert.Equal(t, "test-app", template.Labels["app"])
	assert.Nil(t, template.Annotations)
	assert.Nil(t, template.Type)
}

func TestTargetSecretTemplateAnnotationsOnly(t *testing.T) {
	template := &TargetSecretTemplate{
		Annotations: map[string]string{
			"description": "Test secret",
		},
	}

	assert.Len(t, template.Annotations, 1)
	assert.Equal(t, "Test secret", template.Annotations["description"])
	assert.Nil(t, template.Labels)
	assert.Nil(t, template.Type)
}

func TestTargetSecretTemplateTypeOnly(t *testing.T) {
	secretType := corev1.SecretTypeDockerConfigJson
	template := &TargetSecretTemplate{
		Type: &secretType,
	}

	assert.NotNil(t, template.Type)
	assert.Equal(t, corev1.SecretTypeDockerConfigJson, *template.Type)
	assert.Nil(t, template.Labels)
	assert.Nil(t, template.Annotations)
}

func TestASecretWithCompleteConfiguration(t *testing.T) {
	onlyImport := true
	refreshInterval := metav1.Duration{Duration: 30 * time.Minute}
	secretType := corev1.SecretTypeOpaque

	secret := &ASecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-complete-config",
			Namespace: "production",
		},
		Spec: ASecretSpec{
			TargetSecretName: "app-secrets",
			AwsSecretPath:    "prod/myapp/secrets",
			KmsKeyId:         "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			ValueType:        "json",
			RefreshInterval:  &refreshInterval,
			OnlyImportRemote: &onlyImport,
			Tags: map[string]string{
				"Environment": "production",
				"Application": "myapp",
				"Team":        "backend",
			},
			TargetSecretTemplate: &TargetSecretTemplate{
				Labels: map[string]string{
					"app.kubernetes.io/name":       "myapp",
					"app.kubernetes.io/instance":   "myapp-prod",
					"app.kubernetes.io/component":  "backend",
					"app.kubernetes.io/managed-by": "yet-another-secrets-operator",
				},
				Annotations: map[string]string{
					"description":                  "Production secrets for myapp backend service",
					"owner":                        "backend-team@company.com",
					"secrets.operator/last-update": "2024-01-01T00:00:00Z",
				},
				Type: &secretType,
			},
			Data: map[string]DataSource{
				"database-url": {
					OnlyImportRemote: &onlyImport,
				},
				"api-token": {
					GeneratorRef: &GeneratorReference{
						Name: "api-token-generator",
					},
				},
				"service-account": {
					Value: "backend-service",
				},
			},
		},
	}

	// Test basic fields
	assert.Equal(t, "test-complete-config", secret.Name)
	assert.Equal(t, "production", secret.Namespace)
	assert.Equal(t, "app-secrets", secret.Spec.TargetSecretName)
	assert.Equal(t, "prod/myapp/secrets", secret.Spec.AwsSecretPath)
	assert.Equal(t, "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", secret.Spec.KmsKeyId)
	assert.Equal(t, "json", secret.Spec.ValueType)

	// Test boolean pointers
	assert.NotNil(t, secret.Spec.OnlyImportRemote)
	assert.True(t, *secret.Spec.OnlyImportRemote)

	// Test duration
	assert.NotNil(t, secret.Spec.RefreshInterval)
	assert.Equal(t, 30*time.Minute, secret.Spec.RefreshInterval.Duration)

	// Test tags
	assert.Len(t, secret.Spec.Tags, 3)
	assert.Equal(t, "production", secret.Spec.Tags["Environment"])
	assert.Equal(t, "myapp", secret.Spec.Tags["Application"])
	assert.Equal(t, "backend", secret.Spec.Tags["Team"])

	// Test secret template
	assert.NotNil(t, secret.Spec.TargetSecretTemplate)
	assert.Len(t, secret.Spec.TargetSecretTemplate.Labels, 4)
	assert.Equal(t, "myapp", secret.Spec.TargetSecretTemplate.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "yet-another-secrets-operator", secret.Spec.TargetSecretTemplate.Labels["app.kubernetes.io/managed-by"])

	assert.Len(t, secret.Spec.TargetSecretTemplate.Annotations, 3)
	assert.Contains(t, secret.Spec.TargetSecretTemplate.Annotations["description"], "Production secrets")
	assert.Equal(t, "backend-team@company.com", secret.Spec.TargetSecretTemplate.Annotations["owner"])

	assert.NotNil(t, secret.Spec.TargetSecretTemplate.Type)
	assert.Equal(t, corev1.SecretTypeOpaque, *secret.Spec.TargetSecretTemplate.Type)

	// Test data sources
	assert.Len(t, secret.Spec.Data, 3)

	// Test onlyImportRemote data source
	dbUrl, exists := secret.Spec.Data["database-url"]
	assert.True(t, exists)
	assert.NotNil(t, dbUrl.OnlyImportRemote)
	assert.True(t, *dbUrl.OnlyImportRemote)
	assert.Empty(t, dbUrl.Value)
	assert.Nil(t, dbUrl.GeneratorRef)

	// Test generator ref data source
	apiToken, exists := secret.Spec.Data["api-token"]
	assert.True(t, exists)
	assert.NotNil(t, apiToken.GeneratorRef)
	assert.Equal(t, "api-token-generator", apiToken.GeneratorRef.Name)
	assert.Empty(t, apiToken.Value)
	assert.Nil(t, apiToken.OnlyImportRemote)

	// Test value data source
	serviceAccount, exists := secret.Spec.Data["service-account"]
	assert.True(t, exists)
	assert.Equal(t, "backend-service", serviceAccount.Value)
	assert.Nil(t, serviceAccount.GeneratorRef)
	assert.Nil(t, serviceAccount.OnlyImportRemote)
}
