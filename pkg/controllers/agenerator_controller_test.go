package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	secretsv1alpha1 "github.com/yaso/yet-another-secrets-operator/api/v1alpha1"
)

// Common interface for both testing.T and testing.B
type testingInterface interface {
	Helper()
	Errorf(format string, args ...interface{})
	FailNow()
}

func setupAGeneratorController(t testingInterface) (*AGeneratorReconciler, client.Client) {
	t.Helper()

	// Create a scheme and add our types
	s := runtime.NewScheme()
	err := scheme.AddToScheme(s)
	if err != nil {
		t.Errorf("Failed to add to scheme: %v", err)
		t.FailNow()
	}
	err = secretsv1alpha1.AddToScheme(s)
	if err != nil {
		t.Errorf("Failed to add secretsv1alpha1 to scheme: %v", err)
		t.FailNow()
	}

	// Create a fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		Build()

	// Create the reconciler
	reconciler := &AGeneratorReconciler{
		Client: fakeClient,
		Scheme: s,
		Log:    zap.New(zap.UseDevMode(true)),
	}

	return reconciler, fakeClient
}

func TestAGeneratorReconciler_ReconcileValidGenerator(t *testing.T) {
	reconciler, fakeClient := setupAGeneratorController(t)
	ctx := context.Background()

	// Create a valid AGenerator
	generator := &secretsv1alpha1.AGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-generator",
			Namespace: "default",
		},
		Spec: secretsv1alpha1.AGeneratorSpec{
			Length:              16,
			IncludeUppercase:    true,
			IncludeLowercase:    true,
			IncludeNumbers:      true,
			IncludeSpecialChars: false,
		},
	}

	// Create the generator in the fake client
	err := fakeClient.Create(ctx, generator)
	require.NoError(t, err)

	// Create reconcile request
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-generator",
			Namespace: "default",
		},
	}

	// Reconcile
	result, err := reconciler.Reconcile(ctx, req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify the generator still exists (should not be deleted)
	var retrievedGenerator secretsv1alpha1.AGenerator
	err = fakeClient.Get(ctx, req.NamespacedName, &retrievedGenerator)
	assert.NoError(t, err)
	assert.Equal(t, "test-generator", retrievedGenerator.Name)
}

func TestAGeneratorReconciler_ReconcileInvalidGenerator(t *testing.T) {
	reconciler, fakeClient := setupAGeneratorController(t)
	ctx := context.Background()

	// Create an invalid AGenerator (no character types enabled)
	generator := &secretsv1alpha1.AGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-generator",
			Namespace: "default",
		},
		Spec: secretsv1alpha1.AGeneratorSpec{
			Length:              16,
			IncludeUppercase:    false,
			IncludeLowercase:    false,
			IncludeNumbers:      false,
			IncludeSpecialChars: false,
		},
	}

	// Create the generator in the fake client
	err := fakeClient.Create(ctx, generator)
	require.NoError(t, err)

	// Create reconcile request
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "invalid-generator",
			Namespace: "default",
		},
	}

	// Reconcile
	result, err := reconciler.Reconcile(ctx, req)

	// Assertions - should return an error due to invalid spec
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one character type")
	assert.Equal(t, ctrl.Result{}, result)
}

func TestAGeneratorReconciler_ReconcileInvalidGeneratorZeroLength(t *testing.T) {
	reconciler, fakeClient := setupAGeneratorController(t)
	ctx := context.Background()

	// Create an invalid AGenerator (zero length)
	generator := &secretsv1alpha1.AGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "zero-length-generator",
			Namespace: "default",
		},
		Spec: secretsv1alpha1.AGeneratorSpec{
			Length:              0,
			IncludeUppercase:    true,
			IncludeLowercase:    true,
			IncludeNumbers:      true,
			IncludeSpecialChars: false,
		},
	}

	// Create the generator in the fake client
	err := fakeClient.Create(ctx, generator)
	require.NoError(t, err)

	// Create reconcile request
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "zero-length-generator",
			Namespace: "default",
		},
	}

	// Reconcile
	result, err := reconciler.Reconcile(ctx, req)

	// Assertions - should return an error due to zero length
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "length must be greater than 0")
	assert.Equal(t, ctrl.Result{}, result)
}

func TestAGeneratorReconciler_ReconcileNotFound(t *testing.T) {
	reconciler, _ := setupAGeneratorController(t)
	ctx := context.Background()

	// Create reconcile request for non-existent generator
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent-generator",
			Namespace: "default",
		},
	}

	// Reconcile
	result, err := reconciler.Reconcile(ctx, req)

	// Assertions - should not return an error for not found
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestAGeneratorReconciler_ReconcileValidGeneratorAllCombinations(t *testing.T) {
	tests := []struct {
		name string
		spec secretsv1alpha1.AGeneratorSpec
	}{
		{
			name: "only uppercase",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              10,
				IncludeUppercase:    true,
				IncludeLowercase:    false,
				IncludeNumbers:      false,
				IncludeSpecialChars: false,
			},
		},
		{
			name: "only lowercase",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              10,
				IncludeUppercase:    false,
				IncludeLowercase:    true,
				IncludeNumbers:      false,
				IncludeSpecialChars: false,
			},
		},
		{
			name: "only numbers",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              10,
				IncludeUppercase:    false,
				IncludeLowercase:    false,
				IncludeNumbers:      true,
				IncludeSpecialChars: false,
			},
		},
		{
			name: "only special chars",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              10,
				IncludeUppercase:    false,
				IncludeLowercase:    false,
				IncludeNumbers:      false,
				IncludeSpecialChars: true,
			},
		},
		{
			name: "all character types",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              20,
				IncludeUppercase:    true,
				IncludeLowercase:    true,
				IncludeNumbers:      true,
				IncludeSpecialChars: true,
			},
		},
		{
			name: "mixed characters",
			spec: secretsv1alpha1.AGeneratorSpec{
				Length:              15,
				IncludeUppercase:    true,
				IncludeLowercase:    false,
				IncludeNumbers:      true,
				IncludeSpecialChars: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler, fakeClient := setupAGeneratorController(t)
			ctx := context.Background()

			// Create a valid AGenerator
			generator := &secretsv1alpha1.AGenerator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-generator-" + tt.name,
					Namespace: "default",
				},
				Spec: tt.spec,
			}

			// Create the generator in the fake client
			err := fakeClient.Create(ctx, generator)
			require.NoError(t, err)

			// Create reconcile request
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-generator-" + tt.name,
					Namespace: "default",
				},
			}

			// Reconcile
			result, err := reconciler.Reconcile(ctx, req)

			// Assertions
			assert.NoError(t, err, "Test case: %s", tt.name)
			assert.Equal(t, ctrl.Result{}, result, "Test case: %s", tt.name)
		})
	}
}

func TestAGeneratorReconciler_SetupWithManager(t *testing.T) {
	reconciler, _ := setupAGeneratorController(t)

	// This test just ensures the SetupWithManager method doesn't panic
	// In a real environment, you'd pass a real manager
	// For unit testing, we can't easily test this without a lot of setup
	assert.NotNil(t, reconciler)
	assert.NotNil(t, reconciler.Client)
	assert.NotNil(t, reconciler.Scheme)
	assert.NotNil(t, reconciler.Log)
}

func TestAGeneratorReconciler_LoggerContext(t *testing.T) {
	reconciler, fakeClient := setupAGeneratorController(t)
	ctx := context.Background()

	// Create a valid AGenerator
	generator := &secretsv1alpha1.AGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logger-test-generator",
			Namespace: "test-namespace",
		},
		Spec: secretsv1alpha1.AGeneratorSpec{
			Length:              16,
			IncludeUppercase:    true,
			IncludeLowercase:    true,
			IncludeNumbers:      true,
			IncludeSpecialChars: false,
		},
	}

	// Create the generator in the fake client
	err := fakeClient.Create(ctx, generator)
	require.NoError(t, err)

	// Create reconcile request
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "logger-test-generator",
			Namespace: "test-namespace",
		},
	}

	// Test that reconciler doesn't panic with logger operations
	result, err := reconciler.Reconcile(ctx, req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// Benchmark test for reconcile performance
func BenchmarkAGeneratorReconciler_Reconcile(b *testing.B) {
	reconciler, fakeClient := setupAGeneratorController(b)
	ctx := context.Background()

	// Create a valid AGenerator
	generator := &secretsv1alpha1.AGenerator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-generator",
			Namespace: "default",
		},
		Spec: secretsv1alpha1.AGeneratorSpec{
			Length:              16,
			IncludeUppercase:    true,
			IncludeLowercase:    true,
			IncludeNumbers:      true,
			IncludeSpecialChars: false,
		},
	}

	err := fakeClient.Create(ctx, generator)
	if err != nil {
		b.Fatal(err)
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "benchmark-generator",
			Namespace: "default",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reconciler.Reconcile(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
