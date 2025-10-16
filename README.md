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
    existingSecret:
      onlyImportRemote: true  # Only import from AWS, don't create
    apiKey:
      value: ""
```

## Data Source Options

Each key in the `data` field supports the following options:

- `value`: A hardcoded string value
- `generatorRef`: Reference to an AGenerator for password generation
- `onlyImportRemote`: Boolean flag to only import existing values from AWS without creating new ones

## Secret Template

You can customize the metadata of the generated Kubernetes Secret using the `targetSecretTemplate` field:

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: app-secrets
  namespace: default
spec:
  targetSecretName: my-app-secret
  targetSecretTemplate:
    labels:
      app: my-application
      environment: production
      managed-by: yet-another-secrets-operator
    annotations:
      description: "Application secrets for my-app"
      owner: "platform-team"
    type: Opaque
  awsSecretPath: /my-app/secrets
  data:
    username:
      value: admin
    password:
      generatorRef:
        name: password-generator
```

### Secret Template Options

- `labels`: Custom labels to apply to the Kubernetes Secret
- `annotations`: Custom annotations to apply to the Kubernetes Secret
- `type`: Kubernetes Secret type (e.g., `Opaque`, `kubernetes.io/tls`, `kubernetes.io/dockerconfigjson`)

## How it works

1. The operator checks if the secret exists in AWS Secrets Manager.
2. If the AWS secret exists, it creates or updates the Kubernetes Secret with those values.
3. For keys with `onlyImportRemote: true`, only existing remote values are imported - no new values are created.
4. If there are keys in the ASecret that aren't in the AWS Secret or Kubernetes Secret, they get added (either using the hardcoded value or by generating one).
   1. (optional) if you set removeRemoteKeys, then it'll also remove the remote keys that are not in the ASecret

```markdown README-helm.md
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
    existingApiKey:
      onlyImportRemote: true  # Only import from AWS, don't create if missing
```

## Storing Secrets as Plain JSON

You can store an entire secret as plaintext JSON by setting `valueType: json` in the ASecret spec. This is useful when integrating with IaC tools like Terraform that write secrets as raw JSON.

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: json-app-secret
spec:
  targetSecretName: my-json-app-secret
  awsSecretPath: /my-app/json
  valueType: json
  data:
    json:
      value: |
        {
          "key1": "value1",
          "key2": "value2",
          "list": [1, 2, 3]
        }
```
When `valueType: json`, the operator will treat the secret as a single blob for both synchronize and import.
```

## Storing Binary Data (Certificates, Keys, etc.)

You can store binary data like certificates, private keys, or other binary files by setting `valueType: binary`. This uses AWS Secrets Manager's `SecretBinary` field instead of `SecretString`.

**Important**: Binary secrets can only contain ONE key-value pair in Kubernetes.

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: tls-certificate
  namespace: default
spec:
  targetSecretName: my-tls-cert
  awsSecretPath: /certificates/tls/my-app
  valueType: binary
  targetSecretTemplate:
    type: kubernetes.io/tls
    labels:
      app: my-application
  data:
    # Only ONE key allowed for binary secrets
    tls.crt:
      value: |
        LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t...
        (base64-encoded certificate data)
```

### Import Binary Secrets from AWS

You can import existing binary secrets from AWS Secrets Manager:

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: imported-certificate
  namespace: default
spec:
  targetSecretName: external-cert
  awsSecretPath: /prod/certificates/external
  valueType: binary
  onlyImportRemote: true  # Only import, don't create if missing
  data:
    certificate:  # Key name in the Kubernetes secret
      onlyImportRemote: true
```

When `valueType: binary`:
- The operator uses AWS Secrets Manager's `SecretBinary` field
- Data is automatically base64-encoded when storing in AWS
- Data is automatically decoded when retrieving from AWS
- Only ONE key is allowed in the `data` section
- If no key is specified, defaults to `binaryData`
- Perfect for certificates, keys, and other binary files

### Import-Only Mode

You can configure the operator to only import existing secrets from AWS without creating new ones:

#### Spec-Level Import-Only
When `onlyImportRemote` is set to `true` at the spec level, the operator ignores all data specifications and only imports what exists in AWS:

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: aws-only-secrets
  namespace: default
spec:
  targetSecretName: imported-secret
  awsSecretPath: /existing/aws/secret
  onlyImportRemote: true
  data:
    # This entire data section is ignored when onlyImportRemote=true at spec level
    username:
      value: admin
    password:
      generatorRef:
        name: password-generator
```

### Set Refresh Interval Per Secret

You can specify how often the operator should reconcile a given secret by using the `refreshInterval` field (optional):

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: app-secrets
  namespace: default
spec:
  targetSecretName: my-app-secret
  awsSecretPath: /my-app/secrets
  refreshInterval: 15m
  data:
    username:
      value: admin
```

If not set, the default interval is 1 hour.

## Configuration Options

The following table lists the configurable parameters of the Yet Another Secrets Operator chart:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `ghcr.io/snaax/yet-another-secrets-operator` |
| `image.tag` | Image tag | `0.1.1` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.pullSecrets` | List of image pull secrets | `[]` |
| `replicaCount` | Number of operator replicas | `1` |
| `aws.region` | AWS Region | `` |
| `aws.removeRemoteKeys` | Remove remote keys if not in ASecret | `true` |


## Generate Updated CRDs

After updating the API types, you'll need to regenerate the CRDs using controller-gen or make commands:

```bash
# Typically something like:
make generate
make update-helm-crds
