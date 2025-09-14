# EKS Pod Identity vs IAM Roles for Service Accounts (IRSA)

## Overview

AWS offers two methods for Kubernetes pods to authenticate with AWS services:

1. **EKS Pod Identity** (newer method)
2. **IAM Roles for Service Accounts (IRSA)** (older method)

This document explains the differences between these approaches and how they are used with Another Secrets Operator.

## EKS Pod Identity

EKS Pod Identity is the newer AWS solution that replaces IRSA for EKS authentication. It was launched in November 2023 and provides a simplified and more secure way to access AWS services from EKS pods.

### Key Features

- **No annotations required on ServiceAccounts**
- **Managed by the EKS control plane** - the association between ServiceAccounts and IAM roles is stored in AWS
- **No need for OIDC providers** - simplified setup process
- **Reduced attack surface** - improved security model
- **Works with Fargate and EC2 node groups**

### How It Works

1. You create an IAM role with appropriate permissions
2. You create a Pod Identity Association using the AWS CLI or console
3. The association links an IAM role to a Kubernetes ServiceAccount in a specific namespace
4. When pods using that ServiceAccount run, the EKS control plane automatically provides AWS credentials

### Usage with Another Secrets Operator

```bash
# Create the association
aws eks create-pod-identity-association \
    --cluster-name <your-cluster-name> \
    --namespace system \
    --service-account controller-manager \
    --role-arn arn:aws:iam::<ACCOUNT-ID>:role/another-secrets-operator-role
```

No annotations are needed on the ServiceAccount itself:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: system
  # No annotations needed for Pod Identity
```

## IAM Roles for Service Accounts (IRSA)

IRSA is the older method, introduced in 2019, which uses OIDC federation to provide AWS credentials to pods.

### Key Features

- **Requires annotations on ServiceAccounts** - the role ARN must be specified in the ServiceAccount
- **Requires an OIDC provider** - more complex setup
- **Uses projection volumes** - relies on the AWS SDKs to find credentials in projected files

### How It Works

1. You create an OIDC provider for your EKS cluster
2. You create an IAM role with a trust policy that allows the OIDC provider to assume the role
3. You annotate your ServiceAccount with the role ARN
4. When pods use that ServiceAccount, they receive credentials via projection volumes

### Usage with Another Secrets Operator (IRSA Method)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: controller-manager
  namespace: system
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::<ACCOUNT-ID>:role/another-secrets-operator-role
```

## Which Method Should You Use?

We recommend using **EKS Pod Identity** (when available in your EKS version) because:

1. It has a simpler setup process
2. It's more secure (reduced attack surface)
3. It's AWS's recommended approach going forward
4. It doesn't require annotations on ServiceAccounts
5. It works seamlessly with Another Secrets Operator

Use IRSA only if:
- You're running an older EKS version that doesn't support Pod Identity
- You have existing IRSA infrastructure that you don't want to migrate yet

## Compatibility

Another Secrets Operator is compatible with both authentication methods. The AWS SDK's credential provider chain automatically detects and uses the available authentication method.