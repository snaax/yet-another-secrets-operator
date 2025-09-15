package controllers

import (
	"context" // Add standard errors package
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors" // Rename to avoid conflict
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	secretsv1alpha1 "github.com/yaso/yet-another-secrets-operator/api/v1alpha1"

	awsclient "github.com/yaso/yet-another-secrets-operator/pkg/providers/aws/client"
	"github.com/yaso/yet-another-secrets-operator/pkg/utils"
)

// ASecretReconciler reconciles a ASecret object
type ASecretReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Log            logr.Logger
	AwsClient      *awsclient.AwsClient
	SecretsManager *secretsmanager.Client
}

//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=asecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=asecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=asecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=agenerators,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *ASecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("asecret", req.NamespacedName)
	log.V(1).Info("Reconciling ASecret") // Changed to V(1).Info for less verbose logs

	// Fetch the ASecret instance
	var aSecret secretsv1alpha1.ASecret
	if err := r.Get(ctx, req.NamespacedName, &aSecret); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Use the injected AWS  and client instead of creating a new one
	awsClient := r.AwsClient
	smClient := r.SecretsManager

	// Log which credential provider is being used (useful for debugging)
	if providerName, err := awsClient.GetCredentialProviderInfo(ctx, log); err == nil {
		log.Info("AWS credential provider", "provider", providerName)
	}

	// Check if the secret exists in AWS SecretsManager
	awsSecretData, awsSecretExists, err := r.getAwsSecret(ctx, smClient, aSecret.Spec.AwsSecretPath, log)
	if err != nil {
		log.Error(err, "Failed to check AWS SecretsManager")
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// Look for existing Kubernetes secret
	existingSecret := &corev1.Secret{}
	namespacedName := k8sTypes.NamespacedName{
		Namespace: req.Namespace,
		Name:      aSecret.Spec.TargetSecretName,
	}
	kubeSecretExists := true
	if err := r.Get(ctx, namespacedName, existingSecret); err != nil {
		if apierrors.IsNotFound(err) {
			kubeSecretExists = false
			existingSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      aSecret.Spec.TargetSecretName,
					Namespace: req.Namespace,
				},
			}
		} else {
			log.Error(err, "Failed to get Secret")
			return ctrl.Result{}, err
		}
	}

	// Prepare the secret data
	secretData := make(map[string][]byte)

	// If Kubernetes secret exists, start with its data
	if kubeSecretExists && existingSecret.Data != nil {
		for k, v := range existingSecret.Data {
			secretData[k] = v
		}
	}

	// If AWS secret exists, merge its data (AWS is first source of truth)
	if awsSecretExists {
		for k, v := range awsSecretData {
			secretData[k] = []byte(v)
		}
	}

	// Process ASecret data specifications (last source of truth)
	if err := r.processASecretData(ctx, &aSecret, secretData, log); err != nil {
		log.Error(err, "Failed to process ASecret data")
		return ctrl.Result{}, err
	}

	// Create or update the Kubernetes secret
	if !kubeSecretExists {
		existingSecret.Data = secretData
		existingSecret.Type = corev1.SecretTypeOpaque

		// Set the ASecret as the owner of the Secret
		if err := controllerutil.SetControllerReference(&aSecret, existingSecret, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference on Secret")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, existingSecret); err != nil {
			log.Error(err, "Failed to create Secret")
			return ctrl.Result{}, err
		}
		log.Info("Created Secret", "name", existingSecret.Name)
	} else {
		// Update only if there are changes
		existingSecret.Data = secretData
		if err := r.Update(ctx, existingSecret); err != nil {
			log.Error(err, "Failed to update Secret")
			return ctrl.Result{}, err
		}
		log.Info("Updated Secret", "name", existingSecret.Name)
	}

	// If AWS secret doesn't exist, create it with the current data
	if !awsSecretExists {
		if err := r.createOrUpdateAwsSecret(ctx, smClient, aSecret.Spec.AwsSecretPath, secretData, log); err != nil {
			log.Error(err, "Failed to create AWS Secret")
			return ctrl.Result{}, err
		}
	}

	// Update status
	aSecret.Status.LastSyncTime = metav1.Now()
	aSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Synced",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "ReconciliationSucceeded",
			Message:            "Secret successfully synced",
		},
	}

	if err := r.Status().Update(ctx, &aSecret); err != nil {
		log.Error(err, "Failed to update ASecret status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// getAwsSecret gets a secret from AWS SecretsManager
func (r *ASecretReconciler) getAwsSecret(ctx context.Context, smClient *secretsmanager.Client, secretPath string, log logr.Logger) (map[string]string, bool, error) {
	// Ensure the secret path is formatted correctly
	// AWS expects paths to begin with 'secret/' for hierarchical paths in some cases
	// but we'll use the path as-is and log details if there's a failure
	secretID := secretPath

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	log.Info("Getting AWS secret", "path", secretID)
	result, err := smClient.GetSecretValue(ctx, input)
	if err != nil {
		// Check if it's a resource not found error
		var resourceNotFound *smTypes.ResourceNotFoundException
		if errors.As(err, &resourceNotFound) {
			log.Info("AWS secret not found", "path", secretID)
			return nil, false, nil
		}

		// If it's an endpoint resolution error, log more details
		if err.Error() == "not found, ResolveEndpointV2" {
			log.Error(err, "Failed to resolve AWS endpoint - check AWS region and endpoint configuration",
				"secretPath", secretID,
				"region", os.Getenv("AWS_REGION"))

			return nil, false, fmt.Errorf("AWS endpoint resolution failed for secret %s: %w", secretID, err)
		}

		log.Error(err, "Failed to get AWS secret", "secretPath", secretID)
		return nil, false, err
	}

	// Successfully got the secret, now parse it
	var secretData map[string]string
	if result.SecretString == nil {
		log.Error(nil, "AWS secret value is nil", "secretPath", secretID)
		return nil, true, fmt.Errorf("secret value is nil for %s", secretID)
	}

	if err := json.Unmarshal([]byte(*result.SecretString), &secretData); err != nil {
		log.Error(err, "Failed to unmarshal AWS secret", "secretPath", secretID)
		return nil, true, err
	}

	log.Info("Successfully retrieved AWS secret", "path", secretID, "keys", len(secretData))
	return secretData, true, nil
}

// createOrUpdateAwsSecret creates or updates a secret in AWS SecretsManager
func (r *ASecretReconciler) createOrUpdateAwsSecret(ctx context.Context, smClient *secretsmanager.Client, secretPath string, data map[string][]byte, log logr.Logger) error {
	// Convert binary data to string
	stringData := make(map[string]string)
	for k, v := range data {
		stringData[k] = string(v)
	}

	secretString, err := json.Marshal(stringData)
	if err != nil {
		return err
	}

	// Check if secret exists
	_, err = smClient.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretPath),
	})

	if err != nil {
		// Create new secret if it doesn't exist
		var resourceNotFound *smTypes.ResourceNotFoundException
		if errors.As(err, &resourceNotFound) {
			_, err = smClient.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
				Name:         aws.String(secretPath),
				SecretString: aws.String(string(secretString)),
			})
			return err
		}
		return err
	}

	// Update existing secret
	_, err = smClient.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretPath),
		SecretString: aws.String(string(secretString)),
	})
	return err
}

// processASecretData processes the data from the ASecret, generating values as needed
func (r *ASecretReconciler) processASecretData(ctx context.Context, aSecret *secretsv1alpha1.ASecret, secretData map[string][]byte, log logr.Logger) error {
	for key, dataSource := range aSecret.Spec.Data {
		// Skip if value already exists in the secret data (don't override existing values)
		if _, exists := secretData[key]; exists {
			continue
		}

		// Use hardcoded value if specified
		if dataSource.Value != "" {
			secretData[key] = []byte(dataSource.Value)
			continue
		}

		// Generate value using generator if specified
		if dataSource.GeneratorRef != nil {
			generatedValue, err := r.generateValue(ctx, dataSource.GeneratorRef.Name, log)
			if err != nil {
				return err
			}
			secretData[key] = []byte(generatedValue)
			continue
		}
	}
	return nil
}

// generateValue generates a value using the specified generator
func (r *ASecretReconciler) generateValue(ctx context.Context, generatorName string, log logr.Logger) (string, error) {
	// Get the generator
	var generator secretsv1alpha1.AGenerator
	if err := r.Get(ctx, k8sTypes.NamespacedName{Name: generatorName}, &generator); err != nil {
		log.Error(err, "Failed to get generator", "name", generatorName)
		return "", err
	}

	// Generate value based on generator spec
	value, err := utils.GenerateRandomString(generator.Spec)
	if err != nil {
		return "", err
	}

	return value, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ASecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create the SecretsManager client during setup
	var err error
	ctx := context.Background()
	r.SecretsManager, err = r.AwsClient.CreateSecretsManagerClient(ctx, r.Log)
	if err != nil {
		return fmt.Errorf("failed to create AWS SecretsManager client: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.ASecret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
