# Yet Another Secrets Operator - Helm Chart

This document explains how to deploy Yet Another Secrets Operator using the Helm chart.

## Prerequisites

- Kubernetes 1.16+
- Helm 3+
- AWS account with appropriate permissions
- For EKS Pod Identity: EKS cluster with Pod Identity support

## Installation

### Add the Repository (Optional)

If you are hosting the chart in a repository:

```bash
helm repo add yet-another-secrets-operator https://your-repo-url.com
helm repo update
```

### Install the Chart

From the local chart directory:

```bash
# Change to the chart directory
cd chart/

# Install with default settings
helm install yet-another-secrets-operator ./yet-another-secrets-operator

# Install with custom values
helm install yet-another-secrets-operator ./yet-another-secrets-operator \
  --set image.repository=your-registry/yet-another-secrets-operator \
  --set image.tag=v0.1.0 \
  --set aws.region=us-west-2 \
  --set aws.defaultKmsKeyId=alias/my-secrets-key
```

Or using a values file:

```bash
helm install yet-another-secrets-operator ./yet-another-secrets-operator -f values.yaml
```

## Example Usage

Create an AGenerator:

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

Create an ASecret:

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: app-secrets
  namespace: default
spec:
  targetSecretName: my-app-secret
  awsSecretPath: /my-app/secrets
  data:
    username:
      value: admin
    password:
      generatorRef:
        name: password-generator
```

## Configuration Options

The following table lists the configurable parameters of the Yet Another Secrets Operator chart:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `example/yet-another-secrets-operator` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.pullSecrets` | List of image pull secrets | `[]` |
| `replicaCount` | Number of operator replicas | `1` |
| `aws.region` | AWS Region | `` |
| `aws.removeRemoteKeys` | Remove remote keys if not in ASecret | `false` |
| `aws.defaultKmsKeyId` | Default operator kms key to use | `` |
