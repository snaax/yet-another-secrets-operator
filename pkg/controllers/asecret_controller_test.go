package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secretsv1alpha1 "github.com/yaso/yet-another-secrets-operator/api/v1alpha1"
	awsclient "github.com/yaso/yet-another-secrets-operator/pkg/providers/aws/client"
	"github.com/yaso/yet-another-secrets-operator/pkg/providers/aws/config"
)

// SecretsManagerAPI defines the interface for AWS SecretsManager operations
type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
	TagResource(ctx context.Context, params *secretsmanager.TagResourceInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.TagResourceOutput, error)
}

// MockSecretsManagerClient is a mock implementation of the SecretsManager client
type MockSecretsManagerClient struct {
	mock.Mock
}

func (m *MockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func (m *MockSecretsManagerClient) DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.DescribeSecretOutput), args.Error(1)
}

func (m *MockSecretsManagerClient) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.CreateSecretOutput), args.Error(1)
}

func (m *MockSecretsManagerClient) PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.PutSecretValueOutput), args.Error(1)
}

func (m *MockSecretsManagerClient) TagResource(ctx context.Context, params *secretsmanager.TagResourceInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.TagResourceOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.TagResourceOutput), args.Error(1)
}

func TestApplyTargetSecretTemplate(t *testing.T) {
	tests := []struct {
		name                string
		aSecret             *secretsv1alpha1.ASecret
		secret              *corev1.Secret
		expectedLabels      map[string]string
		expectedAnnotations map[string]string
		expectedType        corev1.SecretType
	}{
		{
			name: "applies all template fields",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
						Labels: map[string]string{
							"app":         "myapp",
							"environment": "production",
						},
						Annotations: map[string]string{
							"description": "Application secrets",
							"owner":       "platform-team",
						},
						Type: &[]corev1.SecretType{corev1.SecretTypeTLS}[0],
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
			},
			expectedLabels: map[string]string{
				"app":         "myapp",
				"environment": "production",
			},
			expectedAnnotations: map[string]string{
				"description": "Application secrets",
				"owner":       "platform-team",
			},
			expectedType: corev1.SecretTypeTLS,
		},
		{
			name: "merges with existing labels and annotations",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
						Labels: map[string]string{
							"app":     "myapp",
							"version": "v2.0",
						},
						Annotations: map[string]string{
							"description": "Updated description",
							"team":        "backend",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
					Labels: map[string]string{
						"app":         "old-app",
						"environment": "staging",
					},
					Annotations: map[string]string{
						"owner":       "old-owner",
						"description": "old description",
					},
				},
			},
			expectedLabels: map[string]string{
				"app":         "myapp",   // overridden
				"environment": "staging", // preserved
				"version":     "v2.0",    // added
			},
			expectedAnnotations: map[string]string{
				"owner":       "old-owner",           // preserved
				"description": "Updated description", // overridden
				"team":        "backend",             // added
			},
			expectedType: corev1.SecretTypeOpaque, // default type
		},
		{
			name: "handles nil template",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: nil,
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
					Labels: map[string]string{
						"existing": "label",
					},
				},
				Type: corev1.SecretTypeDockerConfigJson,
			},
			expectedLabels: map[string]string{
				"existing": "label",
			},
			expectedAnnotations: nil,
			expectedType:        corev1.SecretTypeDockerConfigJson,
		},
		{
			name: "applies only labels",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
						Labels: map[string]string{
							"app": "myapp",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeOpaque,
			},
			expectedLabels: map[string]string{
				"app": "myapp",
			},
			expectedAnnotations: nil,
			expectedType:        corev1.SecretTypeOpaque,
		},
		{
			name: "applies only annotations",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
						Annotations: map[string]string{
							"description": "Test secret",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeOpaque,
			},
			expectedLabels: nil,
			expectedAnnotations: map[string]string{
				"description": "Test secret",
			},
			expectedType: corev1.SecretTypeOpaque,
		},
		{
			name: "applies only type",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
						Type: &[]corev1.SecretType{corev1.SecretTypeServiceAccountToken}[0],
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
					Labels: map[string]string{
						"existing": "label",
					},
				},
				Type: corev1.SecretTypeOpaque,
			},
			expectedLabels: map[string]string{
				"existing": "label",
			},
			expectedAnnotations: nil,
			expectedType:        corev1.SecretTypeServiceAccountToken,
		},
		{
			name: "creates maps when they don't exist",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
						Labels: map[string]string{
							"new": "label",
						},
						Annotations: map[string]string{
							"new": "annotation",
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
			},
			expectedLabels: map[string]string{
				"new": "label",
			},
			expectedAnnotations: map[string]string{
				"new": "annotation",
			},
			expectedType: corev1.SecretTypeOpaque, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			r.applyTargetSecretTemplate(tt.aSecret, tt.secret)

			assert.Equal(t, tt.expectedLabels, tt.secret.Labels)
			assert.Equal(t, tt.expectedAnnotations, tt.secret.Annotations)
			assert.Equal(t, tt.expectedType, tt.secret.Type)
		})
	}
}

func TestPrepareOnlyImportRemoteData(t *testing.T) {
	tests := []struct {
		name            string
		awsSecretData   map[string]string
		awsSecretExists bool
		expected        map[string][]byte
	}{
		{
			name: "AWS secret exists with data",
			awsSecretData: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			awsSecretExists: true,
			expected: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
		},
		{
			name:            "AWS secret doesn't exist",
			awsSecretData:   nil,
			awsSecretExists: false,
			expected:        map[string][]byte{},
		},
		{
			name:            "AWS secret exists but empty",
			awsSecretData:   map[string]string{},
			awsSecretExists: true,
			expected:        map[string][]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			log := logr.Discard()

			result := r.prepareOnlyImportRemoteData(tt.awsSecretData, tt.awsSecretExists, log)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrepareNormalMergeData(t *testing.T) {
	tests := []struct {
		name             string
		aSecret          *secretsv1alpha1.ASecret
		existingSecret   *corev1.Secret
		awsSecretData    map[string]string
		awsSecretExists  bool
		kubeSecretExists bool
		removeRemoteKeys bool
		expected         map[string][]byte
	}{
		{
			name: "merge kube and aws data - aws takes precedence",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {Value: "spec-user"},
						"password": {Value: "spec-pass"},
					},
				},
			},
			existingSecret: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("kube-user"),
					"email":    []byte("kube@example.com"),
				},
			},
			awsSecretData: map[string]string{
				"username": "aws-user",
				"password": "aws-pass",
			},
			awsSecretExists:  true,
			kubeSecretExists: true,
			removeRemoteKeys: false,
			expected: map[string][]byte{
				"username": []byte("aws-user"),
				"email":    []byte("kube@example.com"),
				"password": []byte("aws-pass"),
			},
		},
		{
			name: "prune unmanaged keys",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {Value: "admin"},
					},
				},
			},
			existingSecret: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("kube-user"),
					"old_key":  []byte("should-be-removed"),
				},
			},
			awsSecretData:    map[string]string{},
			awsSecretExists:  false,
			kubeSecretExists: true,
			removeRemoteKeys: true,
			expected: map[string][]byte{
				"username": []byte("kube-user"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{
				AwsClient: &awsclient.AwsClient{
					Config: config.AWSConfig{
						RemoveRemoteKeys: tt.removeRemoteKeys,
					},
				},
			}

			result := r.prepareNormalMergeData(tt.aSecret, tt.existingSecret, tt.awsSecretData, tt.awsSecretExists, tt.kubeSecretExists)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPruneUnmanagedKeys(t *testing.T) {
	tests := []struct {
		name       string
		aSecret    *secretsv1alpha1.ASecret
		secretData map[string][]byte
		expected   map[string][]byte
	}{
		{
			name: "removes unmanaged keys",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {Value: "admin"},
						"password": {Value: "secret"},
					},
				},
			},
			secretData: map[string][]byte{
				"username":    []byte("admin"),
				"password":    []byte("secret"),
				"old_key":     []byte("should-be-removed"),
				"another_old": []byte("also-removed"),
			},
			expected: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
		},
		{
			name: "keeps all keys when all are managed",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {Value: "admin"},
						"password": {Value: "secret"},
					},
				},
			},
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			expected: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
		},
		{
			name: "handles empty managed keys",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{},
				},
			},
			secretData: map[string][]byte{
				"old_key": []byte("should-be-removed"),
			},
			expected: map[string][]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			r.pruneUnmanagedKeys(tt.aSecret, tt.secretData)
			assert.Equal(t, tt.expected, tt.secretData)
		})
	}
}

func TestShouldUpdateAwsSecret(t *testing.T) {
	tests := []struct {
		name            string
		aSecret         *secretsv1alpha1.ASecret
		secretData      map[string][]byte
		awsSecretData   map[string]string
		awsSecretExists bool
		expected        bool
	}{
		{
			name:            "should update when AWS secret doesn't exist",
			aSecret:         &secretsv1alpha1.ASecret{},
			secretData:      map[string][]byte{"key": []byte("value")},
			awsSecretData:   nil,
			awsSecretExists: false,
			expected:        true,
		},
		{
			name:    "should update when keys are missing in AWS",
			aSecret: &secretsv1alpha1.ASecret{},
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			awsSecretData: map[string]string{
				"username": "admin",
			},
			awsSecretExists: true,
			expected:        true,
		},
		{
			name:    "should update when AWS has extra keys",
			aSecret: &secretsv1alpha1.ASecret{},
			secretData: map[string][]byte{
				"username": []byte("admin"),
			},
			awsSecretData: map[string]string{
				"username": "admin",
				"old_key":  "old_value",
			},
			awsSecretExists: true,
			expected:        true,
		},
		{
			name:    "should not update when keys match",
			aSecret: &secretsv1alpha1.ASecret{},
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			awsSecretData: map[string]string{
				"username": "admin",
				"password": "secret",
			},
			awsSecretExists: true,
			expected:        false,
		},
		{
			name: "should filter onlyImportRemote keys",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"apiKey": {
							OnlyImportRemote: boolPtr(true),
						},
					},
				},
			},
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"apiKey":   []byte("secret-key"),
			},
			awsSecretData: map[string]string{
				"username": "admin",
			},
			awsSecretExists: true,
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			result := r.shouldUpdateAwsSecret(tt.aSecret, tt.secretData, tt.awsSecretData, tt.awsSecretExists)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterAwsUpdateData(t *testing.T) {
	tests := []struct {
		name       string
		aSecret    *secretsv1alpha1.ASecret
		secretData map[string][]byte
		expected   map[string][]byte
	}{
		{
			name: "filters out onlyImportRemote keys",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {Value: "admin"},
						"apiKey": {
							Value:            "secret",
							OnlyImportRemote: boolPtr(true),
						},
						"password": {Value: "pass"},
					},
				},
			},
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"apiKey":   []byte("secret"),
				"password": []byte("pass"),
				"other":    []byte("value"),
			},
			expected: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("pass"),
				"other":    []byte("value"),
			},
		},
		{
			name: "keeps all keys when none are onlyImportRemote",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {Value: "admin"},
						"password": {Value: "pass"},
					},
				},
			},
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("pass"),
				"other":    []byte("value"),
			},
			expected: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("pass"),
				"other":    []byte("value"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			result := r.filterAwsUpdateData(tt.aSecret, tt.secretData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipKeyForAwsUpdate(t *testing.T) {
	tests := []struct {
		name     string
		aSecret  *secretsv1alpha1.ASecret
		key      string
		expected bool
	}{
		{
			name: "should skip onlyImportRemote key",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"apiKey": {
							OnlyImportRemote: boolPtr(true),
						},
					},
				},
			},
			key:      "apiKey",
			expected: true,
		},
		{
			name: "should not skip normal key",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {
							Value: "admin",
						},
					},
				},
			},
			key:      "username",
			expected: false,
		},
		{
			name: "should not skip key not in spec",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{},
				},
			},
			key:      "someKey",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			result := r.shouldSkipKeyForAwsUpdate(tt.aSecret, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateKeyDifferences(t *testing.T) {
	tests := []struct {
		name          string
		secretData    map[string][]byte
		awsData       map[string]string
		expectMissing bool
		expectExtra   bool
	}{
		{
			name: "keys missing in AWS",
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			awsData: map[string]string{
				"username": "admin",
			},
			expectMissing: true,
			expectExtra:   false,
		},
		{
			name: "extra keys in AWS",
			secretData: map[string][]byte{
				"username": []byte("admin"),
			},
			awsData: map[string]string{
				"username": "admin",
				"old_key":  "old_value",
			},
			expectMissing: false,
			expectExtra:   true,
		},
		{
			name: "keys match exactly",
			secretData: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			awsData: map[string]string{
				"username": "admin",
				"password": "secret",
			},
			expectMissing: false,
			expectExtra:   false,
		},
		{
			name:          "empty inputs",
			secretData:    map[string][]byte{},
			awsData:       map[string]string{},
			expectMissing: false,
			expectExtra:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			hasMissing, hasExtra := r.calculateKeyDifferences(tt.secretData, tt.awsData)
			assert.Equal(t, tt.expectMissing, hasMissing, "missing keys detection")
			assert.Equal(t, tt.expectExtra, hasExtra, "extra keys detection")
		})
	}
}

func TestParseAwsSecretValue(t *testing.T) {
	tests := []struct {
		name        string
		secretValue string
		valueType   string
		expected    map[string]string
		expectError bool
	}{
		{
			name:        "simple JSON object",
			secretValue: `{"username": "admin", "password": "secret123"}`,
			valueType:   "json",
			expected: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			expectError: false,
		},
		{
			name:        "JSON with mixed types",
			secretValue: `{"username": "admin", "port": 8080, "enabled": true}`,
			valueType:   "json",
			expected: map[string]string{
				"username": "admin",
				"port":     "8080",
				"enabled":  "true",
			},
			expectError: false,
		},
		{
			name:        "JSON with nested object",
			secretValue: `{"config": {"host": "localhost", "port": 8080}}`,
			valueType:   "json",
			expected: map[string]string{
				"config": `{"host":"localhost","port":8080}`,
			},
			expectError: false,
		},
		{
			name:        "legacy format - direct map",
			secretValue: `{"username": "admin", "password": "secret123"}`,
			valueType:   "",
			expected: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			expectError: false,
		},
		{
			name:        "invalid JSON",
			secretValue: `{"username": "admin", "password":}`,
			valueType:   "json",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			result, err := r.parseAwsSecretValue(tt.secretValue, tt.valueType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPrepareAwsSecretString(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string][]byte
		valueType string
		expected  string
	}{
		{
			name: "json valueType preserves types",
			data: map[string][]byte{
				"username": []byte("admin"),
				"port":     []byte("8080"),
				"config":   []byte(`{"nested": true}`),
			},
			valueType: "json",
			expected:  `{"config":{"nested":true},"port":8080,"username":"admin"}`,
		},
		{
			name: "legacy format - all strings",
			data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
			valueType: "",
			expected:  `{"password":"secret123","username":"admin"}`,
		},
		{
			name: "json valueType with invalid JSON becomes string",
			data: map[string][]byte{
				"username":    []byte("admin"),
				"invalidJson": []byte(`{"broken": json`),
			},
			valueType: "json",
			expected:  `{"invalidJson":"{\"broken\": json","username":"admin"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			result, err := r.prepareAwsSecretString(tt.data, tt.valueType)

			require.NoError(t, err)

			// Parse both to compare as objects since JSON key order may vary
			var resultObj, expectedObj interface{}
			err1 := json.Unmarshal([]byte(result), &resultObj)
			err2 := json.Unmarshal([]byte(tt.expected), &expectedObj)

			require.NoError(t, err1)
			require.NoError(t, err2)
			assert.Equal(t, expectedObj, resultObj)
		})
	}
}

func TestPrepareTags(t *testing.T) {
	tests := []struct {
		name        string
		awsClient   *awsclient.AwsClient
		aSecret     *secretsv1alpha1.ASecret
		expectedLen int
		expectTags  map[string]string
	}{
		{
			name: "combines global and secret tags",
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					Tags: map[string]string{
						"managed-by": "yaso",
						"env":        "prod",
					},
				},
			},
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Tags: map[string]string{
						"app":     "myapp",
						"version": "1.0",
					},
				},
			},
			expectedLen: 4,
			expectTags: map[string]string{
				"managed-by": "yaso",
				"env":        "prod",
				"app":        "myapp",
				"version":    "1.0",
			},
		},
		{
			name: "only global tags",
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					Tags: map[string]string{
						"managed-by": "yaso",
					},
				},
			},
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Tags: nil,
				},
			},
			expectedLen: 1,
			expectTags: map[string]string{
				"managed-by": "yaso",
			},
		},
		{
			name: "no tags",
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					Tags: map[string]string{},
				},
			},
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Tags: nil,
				},
			},
			expectedLen: 0,
			expectTags:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{
				AwsClient: tt.awsClient,
			}

			tags := r.prepareTags(tt.aSecret)
			assert.Len(t, tags, tt.expectedLen)

			// Convert tags back to map for easier comparison
			tagMap := make(map[string]string)
			for _, tag := range tags {
				if tag.Key != nil && tag.Value != nil {
					tagMap[*tag.Key] = *tag.Value
				}
			}

			assert.Equal(t, tt.expectTags, tagMap)
		})
	}
}

func TestDetermineKmsKey(t *testing.T) {
	tests := []struct {
		name      string
		aSecret   *secretsv1alpha1.ASecret
		awsClient *awsclient.AwsClient
		expected  string
	}{
		{
			name: "uses ASecret specific KMS key",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					KmsKeyId: "secret-kms-key",
				},
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					DefaultKmsKeyId: "global-kms-key",
				},
			},
			expected: "secret-kms-key",
		},
		{
			name: "uses global default KMS key",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					KmsKeyId: "",
				},
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					DefaultKmsKeyId: "global-kms-key",
				},
			},
			expected: "global-kms-key",
		},
		{
			name: "no KMS key specified",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					KmsKeyId: "",
				},
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					DefaultKmsKeyId: "",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{
				AwsClient: tt.awsClient,
			}
			log := logr.Discard()

			result := r.determineKmsKey(tt.aSecret, log, "test-path")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessASecretData(t *testing.T) {
	tests := []struct {
		name        string
		aSecret     *secretsv1alpha1.ASecret
		secretData  map[string][]byte
		expected    map[string][]byte
		expectError bool
	}{
		{
			name: "hardcoded value gets added when key doesn't exist",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {
							Value: "admin",
						},
					},
				},
			},
			secretData: map[string][]byte{},
			expected: map[string][]byte{
				"username": []byte("admin"),
			},
			expectError: false,
		},
		{
			name: "existing value is not overridden",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {
							Value: "admin",
						},
					},
				},
			},
			secretData: map[string][]byte{
				"username": []byte("existing-user"),
			},
			expected: map[string][]byte{
				"username": []byte("existing-user"),
			},
			expectError: false,
		},
		{
			name: "onlyImportRemote keys are skipped",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"apiKey": {
							Value:            "new-key",
							OnlyImportRemote: boolPtr(true),
						},
						"username": {
							Value: "admin",
						},
					},
				},
			},
			secretData: map[string][]byte{},
			expected: map[string][]byte{
				"username": []byte("admin"),
			},
			expectError: false,
		},
		{
			name: "empty value and no generator skips key",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					Data: map[string]secretsv1alpha1.DataSource{
						"username": {
							Value: "",
						},
					},
				},
			},
			secretData:  map[string][]byte{},
			expected:    map[string][]byte{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			ctx := context.Background()
			log := logr.Discard()

			err := r.processASecretData(ctx, tt.aSecret, tt.secretData, log)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tt.secretData)
			}
		})
	}
}

func TestPrepareSecretData(t *testing.T) {
	tests := []struct {
		name             string
		aSecret          *secretsv1alpha1.ASecret
		existingSecret   *corev1.Secret
		awsSecretData    map[string]string
		awsSecretExists  bool
		kubeSecretExists bool
		removeRemoteKeys bool
		expectOnlyImport bool
	}{
		{
			name: "onlyImportRemote true",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					OnlyImportRemote: boolPtr(true),
				},
			},
			expectOnlyImport: true,
		},
		{
			name: "onlyImportRemote false",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					OnlyImportRemote: boolPtr(false),
				},
			},
			expectOnlyImport: false,
		},
		{
			name: "onlyImportRemote nil (default false)",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					OnlyImportRemote: nil,
				},
			},
			expectOnlyImport: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{
				AwsClient: &awsclient.AwsClient{
					Config: config.AWSConfig{
						RemoveRemoteKeys: tt.removeRemoteKeys,
					},
				},
			}
			log := logr.Discard()

			result := r.prepareSecretData(tt.aSecret, tt.existingSecret, tt.awsSecretData, tt.awsSecretExists, tt.kubeSecretExists, log)

			// Verify that result is not nil
			assert.NotNil(t, result)

			// For onlyImportRemote, the result should only contain AWS data
			if tt.expectOnlyImport {
				if tt.awsSecretExists {
					assert.Len(t, result, len(tt.awsSecretData))
				} else {
					assert.Len(t, result, 0)
				}
			}
		})
	}
}

func TestHandleAwsSecretError(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		secretID          string
		expectedData      map[string]string
		expectedExists    bool
		expectedError     bool
		expectedErrorText string
	}{
		{
			name:           "ResourceNotFoundException",
			err:            &smTypes.ResourceNotFoundException{},
			secretID:       "test-secret",
			expectedData:   nil,
			expectedExists: false,
			expectedError:  false,
		},
		{
			name:              "ResolveEndpointV2 error",
			err:               errors.New("not found, ResolveEndpointV2"),
			secretID:          "test-secret",
			expectedData:      nil,
			expectedExists:    false,
			expectedError:     true,
			expectedErrorText: "AWS endpoint resolution failed for secret test-secret",
		},
		{
			name:              "Generic error",
			err:               errors.New("some other error"),
			secretID:          "test-secret",
			expectedData:      nil,
			expectedExists:    false,
			expectedError:     true,
			expectedErrorText: "some other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			log := logr.Discard()

			data, exists, err := r.handleAwsSecretError(tt.err, tt.secretID, log)

			assert.Equal(t, tt.expectedData, data)
			assert.Equal(t, tt.expectedExists, exists)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedErrorText != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAwsSecret(t *testing.T) {
	tests := []struct {
		name           string
		secret         *secretsv1alpha1.ASecret
		mockResponse   *secretsmanager.GetSecretValueOutput
		mockError      error
		expectedData   map[string]string
		expectedExists bool
		expectedError  bool
	}{
		{
			name: "successful retrieval of JSON secret",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "json",
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"username": "admin", "password": "secret123"}`),
			},
			mockError: nil,
			expectedData: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			expectedExists: true,
			expectedError:  false,
		},
		{
			name: "successful retrieval of legacy format secret",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "",
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"username": "admin", "password": "secret123"}`),
			},
			mockError: nil,
			expectedData: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			expectedExists: true,
			expectedError:  false,
		},
		{
			name: "secret not found",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/nonexistent",
					ValueType:     "json",
				},
			},
			mockResponse:   nil,
			mockError:      &smTypes.ResourceNotFoundException{},
			expectedData:   nil,
			expectedExists: false,
			expectedError:  false,
		},
		{
			name: "nil secret string",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "json",
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretString: nil,
			},
			mockError:      nil,
			expectedData:   nil,
			expectedExists: true,
			expectedError:  true,
		},
		{
			name: "invalid JSON in secret",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "json",
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"invalid": json}`),
			},
			mockError:      nil,
			expectedData:   nil,
			expectedExists: true,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSecretsManagerClient{}
			mockClient.On("GetSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.GetSecretValueInput) bool {
				return *input.SecretId == tt.secret.Spec.AwsSecretPath
			})).Return(tt.mockResponse, tt.mockError)

			r := &ASecretReconciler{}
			ctx := context.Background()
			log := logr.Discard()

			data, exists, err := r.getAwsSecret(ctx, mockClient, tt.secret, log)

			assert.Equal(t, tt.expectedData, data)
			assert.Equal(t, tt.expectedExists, exists)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestCreateAwsSecret(t *testing.T) {
	tests := []struct {
		name          string
		aSecret       *secretsv1alpha1.ASecret
		awsClient     *awsclient.AwsClient
		secretString  string
		tags          []smTypes.Tag
		mockError     error
		expectedError bool
	}{
		{
			name: "successful creation with KMS key",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					KmsKeyId:      "secret-kms-key",
				},
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					DefaultKmsKeyId: "global-kms-key",
				},
			},
			secretString:  `{"username":"admin"}`,
			tags:          []smTypes.Tag{},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "creation fails with AWS error",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					KmsKeyId:      "",
				},
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					DefaultKmsKeyId: "",
				},
			},
			secretString:  `{"username":"admin"}`,
			tags:          []smTypes.Tag{},
			mockError:     errors.New("AWS create error"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSecretsManagerClient{}
			mockClient.On("CreateSecret", mock.Anything, mock.MatchedBy(func(input *secretsmanager.CreateSecretInput) bool {
				return *input.Name == tt.aSecret.Spec.AwsSecretPath && *input.SecretString == tt.secretString
			})).Return(&secretsmanager.CreateSecretOutput{}, tt.mockError)

			r := &ASecretReconciler{
				AwsClient: tt.awsClient,
			}
			ctx := context.Background()
			log := logr.Discard()

			err := r.createAwsSecret(ctx, mockClient, tt.aSecret, tt.secretString, tt.tags, log)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestUpdateAwsSecret(t *testing.T) {
	tests := []struct {
		name             string
		aSecret          *secretsv1alpha1.ASecret
		secretString     string
		tags             []smTypes.Tag
		putSecretError   error
		tagResourceError error
		expectedError    bool
	}{
		{
			name: "successful update with tags",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
				},
			},
			secretString: `{"username":"admin"}`,
			tags: []smTypes.Tag{
				{Key: aws.String("env"), Value: aws.String("test")},
			},
			putSecretError:   nil,
			tagResourceError: nil,
			expectedError:    false,
		},
		{
			name: "successful update without tags",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
				},
			},
			secretString:     `{"username":"admin"}`,
			tags:             []smTypes.Tag{},
			putSecretError:   nil,
			tagResourceError: nil,
			expectedError:    false,
		},
		{
			name: "update fails on PutSecretValue",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
				},
			},
			secretString:     `{"username":"admin"}`,
			tags:             []smTypes.Tag{},
			putSecretError:   errors.New("AWS put error"),
			tagResourceError: nil,
			expectedError:    true,
		},
		{
			name: "update succeeds but tagging fails",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
				},
			},
			secretString: `{"username":"admin"}`,
			tags: []smTypes.Tag{
				{Key: aws.String("env"), Value: aws.String("test")},
			},
			putSecretError:   nil,
			tagResourceError: errors.New("AWS tag error"),
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSecretsManagerClient{}

			mockClient.On("PutSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.PutSecretValueInput) bool {
				return *input.SecretId == tt.aSecret.Spec.AwsSecretPath && *input.SecretString == tt.secretString
			})).Return(&secretsmanager.PutSecretValueOutput{}, tt.putSecretError)

			if len(tt.tags) > 0 && tt.putSecretError == nil {
				mockClient.On("TagResource", mock.Anything, mock.MatchedBy(func(input *secretsmanager.TagResourceInput) bool {
					return *input.SecretId == tt.aSecret.Spec.AwsSecretPath
				})).Return(&secretsmanager.TagResourceOutput{}, tt.tagResourceError)
			}

			r := &ASecretReconciler{}
			ctx := context.Background()

			err := r.updateAwsSecret(ctx, mockClient, tt.aSecret, tt.secretString, tt.tags)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestCreateOrUpdateAwsSecret(t *testing.T) {
	tests := []struct {
		name          string
		aSecret       *secretsv1alpha1.ASecret
		data          map[string][]byte
		awsClient     *awsclient.AwsClient
		describeError error
		createError   error
		updateError   error
		expectedError bool
		expectCreate  bool
	}{
		{
			name: "create new secret when it doesn't exist",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "json",
					Tags: map[string]string{
						"env": "test",
					},
				},
			},
			data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					Tags: map[string]string{
						"managed-by": "yaso",
					},
				},
			},
			describeError: &smTypes.ResourceNotFoundException{},
			createError:   nil,
			updateError:   nil,
			expectedError: false,
			expectCreate:  true,
		},
		{
			name: "update existing secret",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "json",
				},
			},
			data: map[string][]byte{
				"username": []byte("admin"),
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					Tags: map[string]string{},
				},
			},
			describeError: nil,
			createError:   nil,
			updateError:   nil,
			expectedError: false,
			expectCreate:  false,
		},
		{
			name: "describe error other than ResourceNotFound",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/secret",
					ValueType:     "json",
				},
			},
			data: map[string][]byte{
				"username": []byte("admin"),
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{
					Tags: map[string]string{},
				},
			},
			describeError: errors.New("AWS describe error"),
			createError:   errors.New("forced CreateSecret due to describe error"),
			updateError:   nil,
			expectedError: true,
			expectCreate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSecretsManagerClient{}

			mockClient.On("DescribeSecret", mock.Anything, mock.MatchedBy(func(input *secretsmanager.DescribeSecretInput) bool {
				return *input.SecretId == tt.aSecret.Spec.AwsSecretPath
			})).Return(&secretsmanager.DescribeSecretOutput{}, tt.describeError)

			if tt.expectCreate && tt.describeError != nil {
				mockClient.On("CreateSecret", mock.Anything, mock.AnythingOfType("*secretsmanager.CreateSecretInput")).Return(&secretsmanager.CreateSecretOutput{}, tt.createError)
			} else if !tt.expectCreate && tt.describeError == nil {
				mockClient.On("PutSecretValue", mock.Anything, mock.AnythingOfType("*secretsmanager.PutSecretValueInput")).Return(&secretsmanager.PutSecretValueOutput{}, tt.updateError)
				if len(tt.awsClient.Config.Tags) > 0 || len(tt.aSecret.Spec.Tags) > 0 {
					mockClient.On("TagResource", mock.Anything, mock.AnythingOfType("*secretsmanager.TagResourceInput")).Return(&secretsmanager.TagResourceOutput{}, nil)
				}
			}

			r := &ASecretReconciler{
				AwsClient: tt.awsClient,
			}
			ctx := context.Background()
			log := logr.Discard()

			err := r.createOrUpdateAwsSecret(ctx, mockClient, tt.aSecret, tt.data, log)

			if tt.expectCreate {
				mockClient.On("CreateSecret", mock.Anything, mock.AnythingOfType("*secretsmanager.CreateSecretInput")).
					Return(&secretsmanager.CreateSecretOutput{}, tt.createError)
			}

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// Additional test to exercise more branches in parseAwsSecretValue
func TestParseAwsSecretValueErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		secretValue string
		valueType   string
		expectError bool
	}{
		{
			name:        "json type with marshal error fallback",
			secretValue: `{"username": "admin", "complexData": {"nested": "value"}}`,
			valueType:   "json",
			expectError: false,
		},
		{
			name:        "legacy format with invalid JSON",
			secretValue: `invalid json`,
			valueType:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ASecretReconciler{}
			result, err := r.parseAwsSecretValue(tt.secretValue, tt.valueType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// Tests for the new TargetSecretTemplate functionality
func TestApplyTargetSecretTemplateWithTLSSecret(t *testing.T) {
	tlsType := corev1.SecretTypeTLS
	aSecret := &secretsv1alpha1.ASecret{
		Spec: secretsv1alpha1.ASecretSpec{
			TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
				Labels: map[string]string{
					"app.kubernetes.io/name":      "my-app",
					"app.kubernetes.io/instance":  "my-app-prod",
					"app.kubernetes.io/component": "tls",
				},
				Annotations: map[string]string{
					"cert-manager.io/issuer":                   "letsencrypt-prod",
					"cert-manager.io/common-name":              "my-app.example.com",
					"nginx.ingress.kubernetes.io/ssl-redirect": "true",
				},
				Type: &tlsType,
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app-tls",
			Namespace: "production",
		},
		Type: corev1.SecretTypeOpaque,
	}

	r := &ASecretReconciler{}
	r.applyTargetSecretTemplate(aSecret, secret)

	// Verify labels
	assert.Len(t, secret.Labels, 3)
	assert.Equal(t, "my-app", secret.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "my-app-prod", secret.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, "tls", secret.Labels["app.kubernetes.io/component"])

	// Verify annotations
	assert.Len(t, secret.Annotations, 3)
	assert.Equal(t, "letsencrypt-prod", secret.Annotations["cert-manager.io/issuer"])
	assert.Equal(t, "my-app.example.com", secret.Annotations["cert-manager.io/common-name"])
	assert.Equal(t, "true", secret.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"])

	// Verify type was changed
	assert.Equal(t, corev1.SecretTypeTLS, secret.Type)
}

func TestApplyTargetSecretTemplateWithDockerConfigSecret(t *testing.T) {
	dockerConfigType := corev1.SecretTypeDockerConfigJson
	aSecret := &secretsv1alpha1.ASecret{
		Spec: secretsv1alpha1.ASecretSpec{
			TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
				Labels: map[string]string{
					"app.kubernetes.io/name":       "registry-creds",
					"app.kubernetes.io/managed-by": "yet-another-secrets-operator",
				},
				Annotations: map[string]string{
					"description": "Docker registry credentials",
					"registry":    "docker.io",
				},
				Type: &dockerConfigType,
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "docker-registry-creds",
			Namespace: "default",
			Labels: map[string]string{
				"existing": "label",
			},
			Annotations: map[string]string{
				"existing": "annotation",
			},
		},
		Type: corev1.SecretTypeOpaque,
	}

	r := &ASecretReconciler{}
	r.applyTargetSecretTemplate(aSecret, secret)

	// Verify labels were merged
	assert.Len(t, secret.Labels, 3)
	assert.Equal(t, "label", secret.Labels["existing"])
	assert.Equal(t, "registry-creds", secret.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "yet-another-secrets-operator", secret.Labels["app.kubernetes.io/managed-by"])

	// Verify annotations were merged
	assert.Len(t, secret.Annotations, 3)
	assert.Equal(t, "annotation", secret.Annotations["existing"])
	assert.Equal(t, "Docker registry credentials", secret.Annotations["description"])
	assert.Equal(t, "docker.io", secret.Annotations["registry"])

	// Verify type was changed
	assert.Equal(t, corev1.SecretTypeDockerConfigJson, secret.Type)
}

func TestApplyTargetSecretTemplateOverridesExistingMetadata(t *testing.T) {
	serviceAccountType := corev1.SecretTypeServiceAccountToken
	aSecret := &secretsv1alpha1.ASecret{
		Spec: secretsv1alpha1.ASecretSpec{
			TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{
				Labels: map[string]string{
					"app":         "new-app",
					"environment": "production",
				},
				Annotations: map[string]string{
					"description": "Updated description",
					"team":        "platform",
				},
				Type: &serviceAccountType,
			},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-account-token",
			Namespace: "kube-system",
			Labels: map[string]string{
				"app":     "old-app",
				"version": "v1.0",
			},
			Annotations: map[string]string{
				"description": "Old description",
				"owner":       "admin",
			},
		},
		Type: corev1.SecretTypeTLS,
	}

	r := &ASecretReconciler{}
	r.applyTargetSecretTemplate(aSecret, secret)

	// Verify labels: template values override existing, others are preserved
	assert.Len(t, secret.Labels, 3)
	assert.Equal(t, "new-app", secret.Labels["app"])            // overridden
	assert.Equal(t, "v1.0", secret.Labels["version"])           // preserved
	assert.Equal(t, "production", secret.Labels["environment"]) // added

	// Verify annotations: template values override existing, others are preserved
	assert.Len(t, secret.Annotations, 3)
	assert.Equal(t, "Updated description", secret.Annotations["description"]) // overridden
	assert.Equal(t, "admin", secret.Annotations["owner"])                     // preserved
	assert.Equal(t, "platform", secret.Annotations["team"])                   // added

	// Verify type was changed
	assert.Equal(t, corev1.SecretTypeServiceAccountToken, secret.Type)
}

func TestApplyTargetSecretTemplateEmptyTemplate(t *testing.T) {
	aSecret := &secretsv1alpha1.ASecret{
		Spec: secretsv1alpha1.ASecretSpec{
			TargetSecretTemplate: &secretsv1alpha1.TargetSecretTemplate{},
		},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Labels: map[string]string{
				"existing": "label",
			},
			Annotations: map[string]string{
				"existing": "annotation",
			},
		},
	}

	r := &ASecretReconciler{}
	r.applyTargetSecretTemplate(aSecret, secret)

	// Verify nothing changed
	assert.Len(t, secret.Labels, 1)
	assert.Equal(t, "label", secret.Labels["existing"])

	assert.Len(t, secret.Annotations, 1)
	assert.Equal(t, "annotation", secret.Annotations["existing"])

	assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
}

func TestGetAwsSecretBinary(t *testing.T) {
	tests := []struct {
		name           string
		secret         *secretsv1alpha1.ASecret
		mockResponse   *secretsmanager.GetSecretValueOutput
		mockError      error
		expectedData   map[string]string
		expectedExists bool
		expectedError  bool
	}{
		{
			name: "successful retrieval of binary secret with key specified",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
					Data: map[string]secretsv1alpha1.DataSource{
						"tls.crt": {
							OnlyImportRemote: boolPtr(true),
						},
					},
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretBinary: []byte("certificate-data-here"),
			},
			mockError: nil,
			expectedData: map[string]string{
				"tls.crt": "certificate-data-here",
			},
			expectedExists: true,
			expectedError:  false,
		},
		{
			name: "successful retrieval of binary secret without key specified (uses default)",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
					Data:          map[string]secretsv1alpha1.DataSource{},
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretBinary: []byte("certificate-data-here"),
			},
			mockError: nil,
			expectedData: map[string]string{
				"binaryData": "certificate-data-here",
			},
			expectedExists: true,
			expectedError:  false,
		},
		{
			name: "binary secret with multiple keys returns error",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
					Data: map[string]secretsv1alpha1.DataSource{
						"tls.crt": {OnlyImportRemote: boolPtr(true)},
						"tls.key": {OnlyImportRemote: boolPtr(true)},
					},
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretBinary: []byte("certificate-data-here"),
			},
			mockError:      nil,
			expectedData:   nil,
			expectedExists: true,
			expectedError:  true,
		},
		{
			name: "binary secret with nil SecretBinary",
			secret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
				},
			},
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretBinary: nil,
			},
			mockError:      nil,
			expectedData:   nil,
			expectedExists: true,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSecretsManagerClient{}
			mockClient.On("GetSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.GetSecretValueInput) bool {
				return *input.SecretId == tt.secret.Spec.AwsSecretPath
			})).Return(tt.mockResponse, tt.mockError)

			r := &ASecretReconciler{}
			ctx := context.Background()
			log := logr.Discard()

			data, exists, err := r.getAwsSecret(ctx, mockClient, tt.secret, log)

			assert.Equal(t, tt.expectedData, data)
			assert.Equal(t, tt.expectedExists, exists)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestCreateOrUpdateAwsSecretBinary(t *testing.T) {
	tests := []struct {
		name          string
		aSecret       *secretsv1alpha1.ASecret
		data          map[string][]byte
		awsClient     *awsclient.AwsClient
		describeError error
		createError   error
		updateError   error
		expectedError bool
		expectCreate  bool
	}{
		{
			name: "create new binary secret",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
				},
			},
			data: map[string][]byte{
				"tls.crt": []byte("certificate-binary-data"),
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{},
			},
			describeError: &smTypes.ResourceNotFoundException{},
			createError:   nil,
			updateError:   nil,
			expectedError: false,
			expectCreate:  true,
		},
		{
			name: "update existing binary secret",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
				},
			},
			data: map[string][]byte{
				"certificate": []byte("updated-certificate-data"),
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{},
			},
			describeError: nil,
			createError:   nil,
			updateError:   nil,
			expectedError: false,
			expectCreate:  false,
		},
		{
			name: "binary secret with multiple keys returns error",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
				},
			},
			data: map[string][]byte{
				"tls.crt": []byte("cert-data"),
				"tls.key": []byte("key-data"),
			},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{},
			},
			describeError: nil,
			createError:   nil,
			updateError:   nil,
			expectedError: true,
			expectCreate:  false,
		},
		{
			name: "binary secret with no data succeeds (import-only case)",
			aSecret: &secretsv1alpha1.ASecret{
				Spec: secretsv1alpha1.ASecretSpec{
					AwsSecretPath: "/test/cert",
					ValueType:     "binary",
				},
			},
			data: map[string][]byte{},
			awsClient: &awsclient.AwsClient{
				Config: config.AWSConfig{},
			},
			describeError: nil,
			createError:   nil,
			updateError:   nil,
			expectedError: false,
			expectCreate:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockSecretsManagerClient{}

			// Only mock AWS calls for valid cases (1 key or empty data)
			// For multiple keys, the error happens before any AWS calls
			if len(tt.data) == 1 {
				// DescribeSecret is called for valid binary secrets with data
				mockClient.On("DescribeSecret", mock.Anything, mock.MatchedBy(func(input *secretsmanager.DescribeSecretInput) bool {
					return *input.SecretId == tt.aSecret.Spec.AwsSecretPath
				})).Return(&secretsmanager.DescribeSecretOutput{}, tt.describeError)

				if tt.expectCreate && tt.describeError != nil {
					mockClient.On("CreateSecret", mock.Anything, mock.MatchedBy(func(input *secretsmanager.CreateSecretInput) bool {
						return input.SecretBinary != nil
					})).Return(&secretsmanager.CreateSecretOutput{}, tt.createError)
				} else if !tt.expectCreate && tt.describeError == nil {
					mockClient.On("PutSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.PutSecretValueInput) bool {
						return input.SecretBinary != nil
					})).Return(&secretsmanager.PutSecretValueOutput{}, tt.updateError)
				}
			}
			// For empty data (import-only), no AWS calls are made
			// For multiple keys, error is returned before AWS calls

			r := &ASecretReconciler{
				AwsClient: tt.awsClient,
			}
			ctx := context.Background()
			log := logr.Discard()

			err := r.createOrUpdateAwsSecret(ctx, mockClient, tt.aSecret, tt.data, log)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
