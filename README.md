# Another Secrets Operator

Another Secrets Operator (ASO) is a Kubernetes operator that manages secrets between Kubernetes and AWS Secrets Manager. It provides a seamless synchronization between both systems, respecting a defined priority order.

## Features

- Create/Sync Kubernetes Secrets from AWS SecretsManager
- Push Kubernetes Secrets to AWS SecretsManager if they don't exist
- Generate random secret values using configurable generators
- Respect the following priority order for secret values:
  1. AWS Secrets Manager (first source of truth)
  2. Kubernetes Secret (second source of truth)
  3. ASecret CRD (third source of truth)

## Architecture

The operator uses two Custom Resource Definitions (CRDs):

1. **ASecret**: Namespace-scoped resource that defines the secret data, including hardcoded values or references to generators.
2. **AGenerator**: Cluster-wide resource that defines how to generate random values with configurable options.

## Deployment

### Helm Deployment (Recommended)

The recommended way to deploy Another Secrets Operator is using the Helm chart:

```bash
# Build the Docker image first
docker build -t your-registry/another-secrets-operator:latest .
docker push your-registry/another-secrets-operator:latest

# Deploy using Helm
helm install another-secrets-operator ./chart/another-secrets-operator \
  --set image.repository=your-registry/another-secrets-operator \
  --set image.tag=latest
```

For more details on Helm deployment, see [README-helm.md](README-helm.md).

## Usage

### Create a Secret Generator

Create an `AGenerator` to define how random values should be generated:

```yaml
apiVersion: secrets.example.com/v1alpha1
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
apiVersion: secrets.example.com/v1alpha1
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

The operator uses the AWS SDK's default credential provider chain, with built-in support for Pod Identity. You can configure authentication using any of these methods:

### EKS Pod Identity

To use EKS Pod Identity (recommended for EKS clusters):

1. Create an IAM role with appropriate permissions for SecretsManager:

```json
// aws-iam-role.json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret",
        "secretsmanager:PutSecretValue",
        "secretsmanager:CreateSecret",
        "secretsmanager:UpdateSecret",
        "secretsmanager:TagResource"
      ],
      "Resource": [
        "arn:aws:secretsmanager:*:<ACCOUNT-ID>:secret:*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:ListSecrets"
      ],
      "Resource": "*"
    }
  ]
}
```

2. Create an EKS Pod Identity association for your service account:

```bash
eksctl create iamserviceaccount \
  --cluster=<your-cluster> \
  --name=another-secrets-operator \
  --namespace=kube-system \
  --role-name=another-secrets-operator-role \
  --attach-policy-arn=<policy-arn> \
  --approve
```

3. Reference the service account in your operator deployment:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: another-secrets-operator
  namespace: kube-system
spec:
  serviceAccountName: another-secrets-operator  # This links to the Pod Identity-enabled service account
  containers:
    - name: another-secrets-operator
      image: another-secrets-operator:latest
```

### Other Authentication Methods

The operator also supports:

- Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
- AWS credentials file (~/.aws/credentials)
- EC2 Instance Role
- AWS IAM Roles for Service Accounts (IRSA) for older EKS versions

## License

MIT License