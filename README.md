# Yet Another Secrets Operator

## Overview

Yet Another Secrets Operator (YASO) synchronizes secrets between Kubernetes and AWS Secrets Manager, with support for automatic secret generation.

## Custom Resources

### Create an AGenerator

Create an `AGenerator` that defines password generation rules:

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
  specialChars: "!@#$%^&*()-_=+[]{}|;:,.<>?/"
```

### Create an ASecret

Create an `ASecret` that defines your secret:

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
    apiKey:
      value: ""
```

## How it works

1. The operator checks if the secret exists in AWS Secrets Manager.
2. If the AWS secret exists, it creates or updates the Kubernetes Secret with those values.
3. If there are keys in the ASecret that aren't in the AWS Secret or Kubernetes Secret, they get added (either using the hardcoded value or by generating one).
4. If the AWS secret doesn't exist, the operator pushes the Kubernetes Secret data to AWS.
5. The operator maintains sync between both systems, always respecting the priority order.

## Secret Generation Rules

- If a secret value already exists in a Kubernetes Secret, it will never be overwritten.
- When generating values, the operator uses the specified generator's configuration (character sets, length, etc.).
- All generators have sane defaults enabled (uppercase, lowercase, numbers, special chars).

## AWS Authentication

For details on AWS authentication methods supported by this operator, please see our [AWS Authentication Guide](docs/aws-authentication.md).

## Installation

For installation instructions using Helm, please see our [Helm Installation Guide](README-helm.md).
```

### 2. Update examples/asecret-sample-with-namespace.yaml

The example seems consistent with your API group but should be reviewed to ensure it aligns with your implementation:

```yaml examples/asecret-sample-with-namespace.yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: asecret-sample
  namespace: yaso
spec:
  targetSecretName: my-secret
  awsSecretPath: /path/to/aws/secret
  data:
    username:
      value: admin
    password:
      generatorRef:
        name: password-generator
    apiKey:
      value: ""
    dbPassword:
      generatorRef:
        name: strong-password-generator
```

### 3. Update README-helm.md

Update the examples in the Helm README to use the correct API group:

```markdown README-helm.md
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
  --set aws.region=us-west-2
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