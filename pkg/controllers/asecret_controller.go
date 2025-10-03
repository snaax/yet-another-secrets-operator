package controllers

import (
	"context"
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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	SecretsManager awsclient.SecretsManagerAPI
}

//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=asecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=asecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=asecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups=yet-another-secrets.io,resources=agenerators,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *ASecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("asecret", req.NamespacedName)
	log.V(1).Info("Reconciling ASecret")

	// Fetch the ASecret instance
	var aSecret secretsv1alpha1.ASecret
	if err := r.Get(ctx, req.NamespacedName, &aSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Use the injected AWS client
	awsClient := r.AwsClient
	smClient := r.SecretsManager

	// Log which credential provider is being used
	if providerName, err := awsClient.GetCredentialProviderInfo(ctx, log); err == nil {
		log.V(1).Info("AWS credential provider", "provider", providerName)
	}

	// Check if the secret exists in AWS SecretsManager
	awsSecretData, awsSecretExists, err := r.getAwsSecret(ctx, smClient, &aSecret, log)
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

	// Prepare the secret data using extracted function
	secretData := r.prepareSecretData(&aSecret, existingSecret, awsSecretData, awsSecretExists, kubeSecretExists, log)

	// Process ASecret data specifications if not onlyImportRemote
	onlyImportRemote := aSecret.Spec.OnlyImportRemote != nil && *aSecret.Spec.OnlyImportRemote
	if !onlyImportRemote {
		if err := r.processASecretData(ctx, &aSecret, secretData, log); err != nil {
			log.Error(err, "Failed to process ASecret data")
			return ctrl.Result{}, err
		}
	}

	// Create or update the Kubernetes secret
	if !kubeSecretExists {
		existingSecret.Data = secretData
		existingSecret.Type = corev1.SecretTypeOpaque

		// Apply target secret template if specified
		r.applyTargetSecretTemplate(&aSecret, existingSecret)

		if err := controllerutil.SetControllerReference(&aSecret, existingSecret, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference on Secret")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, existingSecret); err != nil {
			log.Error(err, "Failed to create Secret")
			return ctrl.Result{}, err
		}
		log.Info("Created Kubernetes Secret", "name", existingSecret.Name)
	} else {
		existingSecret.Data = secretData

		// Apply target secret template if specified
		r.applyTargetSecretTemplate(&aSecret, existingSecret)

		if err := r.Update(ctx, existingSecret); err != nil {
			log.Error(err, "Failed to update Secret")
			return ctrl.Result{}, err
		}
		log.Info("Updated Kubernetes Secret", "name", existingSecret.Name)
	}

	// Update AWS secret if needed
	if !onlyImportRemote {
		needsUpdate := r.shouldUpdateAwsSecret(&aSecret, secretData, awsSecretData, awsSecretExists)
		if needsUpdate {
			if err := r.createOrUpdateAwsSecret(ctx, smClient, &aSecret, secretData, log); err != nil {
				log.Error(err, "Failed to create AWS Secret")
				return ctrl.Result{}, err
			}
			log.Info("Updated AWS Secret", "name", existingSecret.Name)
		}
	} else {
		log.V(1).Info("OnlyImportRemote set, nothing updated on AWS Secret", "name", existingSecret.Name)
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

	// Compute per-secret refresh interval (defaults to 1h if not set)
	requeue := time.Hour
	if aSecret.Spec.RefreshInterval != nil && aSecret.Spec.RefreshInterval.Duration > 0 {
		requeue = aSecret.Spec.RefreshInterval.Duration
	}

	return ctrl.Result{RequeueAfter: requeue}, nil
}

// prepareSecretData handles the logic for preparing secret data from various sources
func (r *ASecretReconciler) prepareSecretData(aSecret *secretsv1alpha1.ASecret, existingSecret *corev1.Secret, awsSecretData map[string]string, awsSecretExists, kubeSecretExists bool, log logr.Logger) map[string][]byte {
	onlyImportRemote := aSecret.Spec.OnlyImportRemote != nil && *aSecret.Spec.OnlyImportRemote

	if onlyImportRemote {
		return r.prepareOnlyImportRemoteData(awsSecretData, awsSecretExists, log)
	}

	return r.prepareNormalMergeData(aSecret, existingSecret, awsSecretData, awsSecretExists, kubeSecretExists)
}

// applyTargetSecretTemplate applies the secret template configuration to the Kubernetes Secret
func (r *ASecretReconciler) applyTargetSecretTemplate(aSecret *secretsv1alpha1.ASecret, secret *corev1.Secret) {
	if aSecret.Spec.TargetSecretTemplate == nil {
		return
	}

	template := aSecret.Spec.TargetSecretTemplate

	// Apply labels
	if template.Labels != nil {
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		for k, v := range template.Labels {
			secret.Labels[k] = v
		}
	}

	// Apply annotations
	if template.Annotations != nil {
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		for k, v := range template.Annotations {
			secret.Annotations[k] = v
		}
	}

	// Apply secret type
	if template.Type != nil {
		secret.Type = *template.Type
	} else {
		secret.Type = corev1.SecretTypeOpaque
	}
}

// prepareOnlyImportRemoteData prepares data when onlyImportRemote is true
func (r *ASecretReconciler) prepareOnlyImportRemoteData(awsSecretData map[string]string, awsSecretExists bool, log logr.Logger) map[string][]byte {
	log.Info("OnlyImportRemote enabled - importing only from AWS")
	secretData := make(map[string][]byte)

	if awsSecretExists {
		for k, v := range awsSecretData {
			secretData[k] = []byte(v)
		}
		log.Info("Imported data from AWS", "keys", len(secretData))
	} else {
		log.Info("No AWS secret found and OnlyImportRemote is true - creating empty secret")
	}

	return secretData
}

// prepareNormalMergeData prepares data with normal merging logic
func (r *ASecretReconciler) prepareNormalMergeData(aSecret *secretsv1alpha1.ASecret, existingSecret *corev1.Secret, awsSecretData map[string]string, awsSecretExists, kubeSecretExists bool) map[string][]byte {
	secretData := make(map[string][]byte)

	// Start with Kubernetes secret data if it exists
	if kubeSecretExists && existingSecret.Data != nil {
		for k, v := range existingSecret.Data {
			secretData[k] = v
		}
	}

	// AWS data takes precedence
	if awsSecretExists {
		for k, v := range awsSecretData {
			secretData[k] = []byte(v)
		}
	}

	// Apply key pruning if configured
	if r.AwsClient.Config.RemoveRemoteKeys {
		r.pruneUnmanagedKeys(aSecret, secretData)
	}

	return secretData
}

// pruneUnmanagedKeys removes keys that are no longer managed by the ASecret
func (r *ASecretReconciler) pruneUnmanagedKeys(aSecret *secretsv1alpha1.ASecret, secretData map[string][]byte) {
	managedKeys := make(map[string]bool)
	for key := range aSecret.Spec.Data {
		managedKeys[key] = true
	}

	keysToDelete := []string{}
	for k := range secretData {
		if !managedKeys[k] {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, k := range keysToDelete {
		delete(secretData, k)
	}
}

// shouldUpdateAwsSecret determines if AWS secret needs to be updated
func (r *ASecretReconciler) shouldUpdateAwsSecret(aSecret *secretsv1alpha1.ASecret, secretData map[string][]byte, awsSecretData map[string]string, awsSecretExists bool) bool {
	if !awsSecretExists {
		return true
	}

	// Prepare data for AWS update, excluding onlyImportRemote keys
	awsUpdateData := r.filterAwsUpdateData(aSecret, secretData)

	// Check for differences
	hasMissingKeys, hasExtraKeys := r.calculateKeyDifferences(awsUpdateData, awsSecretData)
	return hasMissingKeys || hasExtraKeys
}

// filterAwsUpdateData filters out keys that shouldn't be written to AWS
func (r *ASecretReconciler) filterAwsUpdateData(aSecret *secretsv1alpha1.ASecret, secretData map[string][]byte) map[string][]byte {
	awsUpdateData := make(map[string][]byte)
	for k, v := range secretData {
		if !r.shouldSkipKeyForAwsUpdate(aSecret, k) {
			awsUpdateData[k] = v
		}
	}
	return awsUpdateData
}

// shouldSkipKeyForAwsUpdate checks if a key should be skipped for AWS updates
func (r *ASecretReconciler) shouldSkipKeyForAwsUpdate(aSecret *secretsv1alpha1.ASecret, key string) bool {
	if dataSource, exists := aSecret.Spec.Data[key]; exists &&
		dataSource.OnlyImportRemote != nil && *dataSource.OnlyImportRemote {
		return true
	}
	return false
}

// calculateKeyDifferences checks for missing or extra keys between local and AWS data
func (r *ASecretReconciler) calculateKeyDifferences(secretData map[string][]byte, awsSecretData map[string]string) (bool, bool) {
	// Check for keys that exist in secretData but not in AWS
	hasMissingKeys := false
	for k := range secretData {
		if _, exists := awsSecretData[k]; !exists {
			hasMissingKeys = true
			break
		}
	}

	// Check for keys that exist in AWS but not in secretData
	hasExtraKeys := false
	for k := range awsSecretData {
		if _, exists := secretData[k]; !exists {
			hasExtraKeys = true
			break
		}
	}

	return hasMissingKeys, hasExtraKeys
}

// getAwsSecret gets a secret from AWS SecretsManager
func (r *ASecretReconciler) getAwsSecret(ctx context.Context, smClient awsclient.SecretsManagerAPI, secret *secretsv1alpha1.ASecret, log logr.Logger) (map[string]string, bool, error) {
	secretID := secret.Spec.AwsSecretPath
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}

	log.V(1).Info("Getting AWS secret", "path", secretID)
	result, err := smClient.GetSecretValue(ctx, input)
	if err != nil {
		return r.handleAwsSecretError(err, secretID, log)
	}

	if result.SecretString == nil {
		log.Error(nil, "AWS secret value is nil", "secretPath", secretID)
		return nil, true, fmt.Errorf("secret value is nil for %s", secretID)
	}

	secretData, err := r.parseAwsSecretValue(*result.SecretString, secret.Spec.ValueType)
	if err != nil {
		log.Error(err, "Failed to unmarshal AWS secret", "secretPath", secretID)
		return nil, true, err
	}

	log.V(1).Info("Successfully retrieved AWS secret", "path", secretID, "keys", len(secretData))
	return secretData, true, nil
}

// handleAwsSecretError handles errors from AWS SecretsManager operations
func (r *ASecretReconciler) handleAwsSecretError(err error, secretID string, log logr.Logger) (map[string]string, bool, error) {
	var resourceNotFound *smTypes.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		log.Info("AWS secret not found", "path", secretID)
		return nil, false, nil
	}

	if err.Error() == "not found, ResolveEndpointV2" {
		log.Error(err, "Failed to resolve AWS endpoint - check AWS region and endpoint configuration",
			"secretPath", secretID,
			"region", os.Getenv("AWS_REGION"))
		return nil, false, fmt.Errorf("AWS endpoint resolution failed for secret %s: %w", secretID, err)
	}

	log.Error(err, "Failed to get AWS secret", "secretPath", secretID)
	return nil, false, err
}

// parseAwsSecretValue parses the AWS secret value based on the valueType
func (r *ASecretReconciler) parseAwsSecretValue(secretValue, valueType string) (map[string]string, error) {
	if valueType == "json" {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(secretValue), &obj); err != nil {
			return nil, err
		}

		secretData := make(map[string]string)
		for k, v := range obj {
			switch t := v.(type) {
			case string:
				secretData[k] = t
			default:
				bytes, err := json.Marshal(t)
				if err != nil {
					secretData[k] = fmt.Sprintf("%v", t)
				} else {
					secretData[k] = string(bytes)
				}
			}
		}
		return secretData, nil
	}

	var secretData map[string]string
	if err := json.Unmarshal([]byte(secretValue), &secretData); err != nil {
		return nil, err
	}
	return secretData, nil
}

// createOrUpdateAwsSecret creates or updates a secret in AWS SecretsManager
func (r *ASecretReconciler) createOrUpdateAwsSecret(ctx context.Context, smClient awsclient.SecretsManagerAPI, aSecret *secretsv1alpha1.ASecret, data map[string][]byte, log logr.Logger) error {
	secretString, err := r.prepareAwsSecretString(data, aSecret.Spec.ValueType)
	if err != nil {
		return err
	}

	secretPath := aSecret.Spec.AwsSecretPath

	// Check if secret exists
	_, err = smClient.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretPath),
	})

	tags := r.prepareTags(aSecret)

	if err != nil {
		return r.createAwsSecret(ctx, smClient, aSecret, secretString, tags, log)
	}

	return r.updateAwsSecret(ctx, smClient, aSecret, secretString, tags)
}

// prepareAwsSecretString prepares the secret string for AWS
func (r *ASecretReconciler) prepareAwsSecretString(data map[string][]byte, valueType string) (string, error) {
	if valueType == "json" {
		obj := make(map[string]interface{})
		for k, v := range data {
			var vObj interface{}
			if json.Unmarshal(v, &vObj) == nil {
				obj[k] = vObj
			} else {
				obj[k] = string(v)
			}
		}
		secretString, err := json.Marshal(obj)
		return string(secretString), err
	}

	// legacy: marshal as object/map
	stringData := make(map[string]string)
	for k, v := range data {
		stringData[k] = string(v)
	}
	secretString, err := json.Marshal(stringData)
	return string(secretString), err
}

// prepareTags prepares AWS tags from config and ASecret spec
func (r *ASecretReconciler) prepareTags(aSecret *secretsv1alpha1.ASecret) []smTypes.Tag {
	var tags []smTypes.Tag

	// Add global config tags
	for k, v := range r.AwsClient.Config.Tags {
		tags = append(tags, smTypes.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// Add ASecret spec tags
	if aSecret.Spec.Tags != nil {
		for k, v := range aSecret.Spec.Tags {
			tags = append(tags, smTypes.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
	}

	return tags
}

// createAwsSecret creates a new AWS secret
func (r *ASecretReconciler) createAwsSecret(ctx context.Context, smClient awsclient.SecretsManagerAPI, aSecret *secretsv1alpha1.ASecret, secretString string, tags []smTypes.Tag, log logr.Logger) error {
	secretPath := aSecret.Spec.AwsSecretPath
	createInput := &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretPath),
		SecretString: aws.String(secretString),
		Tags:         tags,
	}

	// Determine KMS key
	kmsKeyId := r.determineKmsKey(aSecret, log, secretPath)
	if kmsKeyId != "" {
		createInput.KmsKeyId = aws.String(kmsKeyId)
	} else {
		log.V(1).Info("Creating AWS secret with default encryption", "path", secretPath)
	}

	_, err := smClient.CreateSecret(ctx, createInput)
	return err
}

// updateAwsSecret updates an existing AWS secret
func (r *ASecretReconciler) updateAwsSecret(ctx context.Context, smClient awsclient.SecretsManagerAPI, aSecret *secretsv1alpha1.ASecret, secretString string, tags []smTypes.Tag) error {
	secretPath := aSecret.Spec.AwsSecretPath

	// Update secret value
	_, err := smClient.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretPath),
		SecretString: aws.String(secretString),
	})

	// Update tags if no error and tags exist
	if err == nil && len(tags) > 0 {
		_, err = smClient.TagResource(ctx, &secretsmanager.TagResourceInput{
			SecretId: aws.String(secretPath),
			Tags:     tags,
		})
	}

	return err
}

// determineKmsKey determines which KMS key to use
func (r *ASecretReconciler) determineKmsKey(aSecret *secretsv1alpha1.ASecret, log logr.Logger, secretPath string) string {
	if aSecret.Spec.KmsKeyId != "" {
		log.V(1).Info("Using ASecret-specific KMS key", "path", secretPath, "kmsKeyId", aSecret.Spec.KmsKeyId)
		return aSecret.Spec.KmsKeyId
	}

	if r.AwsClient.Config.DefaultKmsKeyId != "" {
		log.V(1).Info("Using global default KMS key", "path", secretPath, "kmsKeyId", r.AwsClient.Config.DefaultKmsKeyId)
		return r.AwsClient.Config.DefaultKmsKeyId
	}

	return ""
}

// processASecretData processes the data from the ASecret, generating values as needed
func (r *ASecretReconciler) processASecretData(ctx context.Context, aSecret *secretsv1alpha1.ASecret, secretData map[string][]byte, log logr.Logger) error {
	for key, dataSource := range aSecret.Spec.Data {
		if dataSource.OnlyImportRemote != nil && *dataSource.OnlyImportRemote {
			log.V(1).Info("Skipping key with onlyImportRemote=true", "key", key)
			continue
		}

		if _, exists := secretData[key]; exists {
			continue
		}

		if dataSource.Value != "" {
			secretData[key] = []byte(dataSource.Value)
			continue
		}

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
	var generator secretsv1alpha1.AGenerator
	if err := r.Get(ctx, k8sTypes.NamespacedName{Name: generatorName}, &generator); err != nil {
		log.Error(err, "Failed to get generator", "name", generatorName)
		return "", err
	}

	value, err := utils.GenerateRandomString(generator.Spec)
	if err != nil {
		return "", err
	}

	return value, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ASecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
