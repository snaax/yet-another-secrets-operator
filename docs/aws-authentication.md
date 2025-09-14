# AWS Authentication Guide

This document explains how to configure AWS authentication for the Another Secrets Operator, with special focus on EKS Pod Identity.

## AWS Authentication Methods

The operator supports multiple methods for authenticating with AWS:

1. EKS Pod Identity (recommended for EKS clusters)
2. IAM Roles for Service Accounts (IRSA) (for older EKS clusters)
3. Environment variables
4. Shared credentials file
5. EC2 Instance Profile

## EKS Pod Identity Setup

### Prerequisites

- An EKS cluster with Pod Identity enabled
- AWS CLI configured with appropriate permissions
- kubectl configured to work with your EKS cluster

### Step 1: Create IAM Policy

Create an IAM policy that grants the necessary permissions to work with AWS Secrets Manager:

```bash
# Create policy file
cat > secretsmanager-policy.json << EOF
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
EOF

# Create the policy
aws iam create-policy \
    --policy-name ASOSecretsManagerPolicy \
    --policy-document file://secretsmanager-policy.json
```

### Step 2: Create IAM Role for Pod Identity

```bash
# Create the IAM role for Pod Identity
aws iam create-role \
    --role-name another-secrets-operator-role \
    --assume-role-policy-document '{
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "pods.eks.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }'

# Attach the policy to the role
aws iam attach-role-policy \
    --role-name another-secrets-operator-role \
    --policy-arn arn:aws:iam::<ACCOUNT-ID>:policy/ASOSecretsManagerPolicy
```

### Step 3: Create Pod Identity Association

```bash
# Create the Pod Identity Association
aws eks create-pod-identity-association \
    --cluster-name <your-cluster-name> \
    --namespace yaso \
    --service-account another-secrets-operator \
    --role-arn arn:aws:iam::<ACCOUNT-ID>:role/another-secrets-operator-role
```

Note: With EKS Pod Identity, you no longer need to add annotations to the ServiceAccount. The association is managed by AWS EKS directly.

### Step 4: Deploy the Operator

Update the operator's deployment to use the ServiceAccount associated with Pod Identity:

```bash
# Apply the configuration
kubectl apply -f config/manager/manager.yaml
```

## Environment Configuration

The operator supports the following environment variables for AWS configuration:

| Variable | Description |
|----------|-------------|
| `AWS_REGION` | AWS region to use |
| `AWS_ENDPOINT_URL` | Custom endpoint URL (for testing) |
| `AWS_ACCESS_KEY_ID` | AWS access key (not recommended, use Pod Identity) |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key (not recommended, use Pod Identity) |
| `AWS_SESSION_TOKEN` | AWS session token (not recommended, use Pod Identity) |

These can be set in the deployment manifest:

```yaml
spec:
  containers:
  - name: manager
    env:
    - name: AWS_REGION
      value: "us-west-2"
```

## Troubleshooting

### Verifying Identity Configuration

To check which credential provider is being used by the operator, look at the logs:

```bash
kubectl logs -n system deployment/controller-manager
```

Look for log entries like: `AWS credential provider="EKSPodIdentityCredentialProvider"`

### Common Issues

1. **Permission Denied Errors**:
   - Check that the IAM policy attached to the role has the correct permissions
   - Verify the Pod Identity association was created successfully

2. **Role Not Found**:
   - Ensure the IAM role exists and has the correct trust relationship
   - Check that the EKS cluster has Pod Identity enabled

3. **Region Issues**:
   - Set the `AWS_REGION` environment variable to match your SecretsManager resources
   - Verify that the resources exist in the specified region

4. **Pod Identity Not Working**:
   - Ensure your EKS version supports Pod Identity
   - Check that the ServiceAccount name and namespace match the Pod Identity association