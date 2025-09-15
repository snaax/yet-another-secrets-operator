# Yet Another Secrets Operator - Helm Chart

This document explains how to deploy Yet Another Secrets Operator using the Helm chart.

## Prerequisites

- Kubernetes 1.16+
- Helm 3+
- AWS account with appropriate permissions
- For EKS Pod Identity: EKS cluster with Pod Identity support

## Installation

### Add the Repository

```bash
helm repo add yet-another-secrets-operator https://snaax.github.io/yet-another-secrets-operator/charts
helm repo update
```

### Install the Chart

From the repository:

```bash
# Install with default settings
helm install yaso yet-another-secrets-operator/yet-another-secrets-operator

# Install with custom values
helm install yaso yet-another-secrets-operator/yet-another-secrets-operator \
  --set image.repository=your-registry/yet-another-secrets-operator \
  --set image.tag=v0.1.0 \
  --set aws.region=us-west-2
```

Or from the local chart directory:

```bash
# Change to the chart directory
cd chart/

# Install with default settings
helm install yaso ./yet-another-secrets-operator

# Install with custom values
helm install yaso ./yet-another-secrets-operator \
  --set image.repository=your-registry/yet-another-secrets-operator \
  --set image.tag=v0.1.0 \
  --set aws.region=us-west-2
```

Or using a values file:

```bash
helm install yaso ./yet-another-secrets-operator -f values.yaml
```

## Using as a Dependency in Your Chart

To include Yet Another Secrets Operator as a dependency in your Helm chart:

```yaml
apiVersion: v2
name: your-application
description: Your application description
type: application
version: 0.1.0
dependencies:
- name: yet-another-secrets-operator
  version: "0.1.0"
  repository: "https://snaax.github.io/yet-another-secrets-operator/charts"
```

Then, update your dependencies:

```bash
helm dependency update
```

## Configuration Options

The following table lists the configurable parameters of the Yet Another Secrets Operator chart:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `ghcr.io/snaax/yet-another-secrets-operator` |
| `image.tag` | Image tag | `0.1.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.pullSecrets` | List of image pull secrets | `[]` |
| `replicaCount` | Number of operator replicas | `1` |
| `namespace` | Namespace to install the operator in | `yaso` |
```

### 2. Update chart/yet-another-secrets-operator/templates/NOTES.txt

```
Thank you for installing {{ .Chart.Name }} version {{ .Chart.Version }}.

The Yet Another Secrets Operator has been deployed to your cluster in the {{ include "yet-another-secrets-operator.namespace" . }} namespace.

## Next Steps

1. Create an AGenerator for password generation:

   ```yaml
   apiVersion: yet-another-secrets.io/v1alpha1
   kind: AGenerator
   metadata:
     name: password-generator
   spec:
     length: 16
     includeUppercase: true
     includeLowercase: true
     includeNumbers: true
     includeSpecialChars: true
   ```

2. Create an ASecret that references your generator:

   ```yaml
   apiVersion: yet-another-secrets.io/v1alpha1
   kind: ASecret
   metadata:
     name: example-secret
     namespace: {{ include "yet-another-secrets-operator.namespace" . }}
   spec:
     targetSecretName: my-app-secret
     awsSecretPath: /path/to/aws/secret
     data:
       username:
         value: admin
       password:
         generatorRef:
           name: password-generator
   ```

3. Verify the operator created your secret:

   ```bash
   kubectl get secret my-app-secret -n {{ include "yet-another-secrets-operator.namespace" . }}
   ```

NOTE: Before using the operator, make sure Pod Identity is configured for your EKS cluster:

```bash
aws eks create-pod-identity-association \
    --cluster-name <your-cluster-name> \
    --namespace {{ include "yet-another-secrets-operator.namespace" . }} \
    --role-arn <your-role-arn> \
    --service-account {{ include "yet-another-secrets-operator.serviceAccountName" . }}
```
```

### 3. Create an Example Dependency Chart (docs/examples/test-chart/Chart.yaml)

```yaml docs/examples/test-chart/Chart.yaml
apiVersion: v2
name: test-yaso
description: YASO test
type: application
version: 0.0.1
dependencies:
- name: yet-another-secrets-operator
  version: "0.1.0"
  repository: "https://snaax.github.io/yet-another-secrets-operator/charts"
```

### 4. Update CI/CD workflow file

In your `.github/workflows/build-and-publish.yaml` file, ensure consistency in references:

```yaml .github/workflows/build-and-publish.yaml
# Update values.yaml with the correct image reference
sed -i "s|repository: example/yet-another-secrets-operator|repository: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}|g" chart/yet-another-secrets-operator/values.yaml
sed -i "s|tag: latest|tag: ${{ steps.chart_version.outputs.app_version }}|g" chart/yet-another-secrets-operator/values.yaml