# Another Secrets Operator - Helm Chart

This document explains how to deploy Another Secrets Operator using the Helm chart.

## Prerequisites

- Kubernetes 1.16+
- Helm 3+
- AWS account with appropriate permissions
- For EKS Pod Identity: EKS cluster with Pod Identity support

## Installation

### Add the Repository (Optional)

If you are hosting the chart in a repository:

```bash
helm repo add another-secrets-operator https://your-repo-url.com
helm repo update
```

### Install the Chart

From the local chart directory:

```bash
# Change to the chart directory
cd chart/

# Install with default settings
helm install another-secrets-operator ./another-secrets-operator

# Install with custom values
helm install another-secrets-operator ./another-secrets-operator \
  --set image.repository=your-registry/another-secrets-operator \
  --set image.tag=v0.1.0 \
  --set aws.region=us-west-2
```

Or using a values file:

```bash
helm install another-secrets-operator ./another-secrets-operator -f values.yaml
```

## Configuration Options

The following table lists the configurable parameters of the Another Secrets Operator chart:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Image repository | `example/another-secrets-operator` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.pullSecrets` | List of image pull secrets | `[]` |
| `replicaCount` | Number of operator replicas | `1` |
| `namespace` | Namespace to install the operator | `yaso` |
| `serviceAccount.name` | Name of the service account | `another-secrets-operator` |

| `aws.region` | AWS region to use | `""` (uses Pod's environment) |

| `resources` | CPU/Memory resource requests/limits | Requests: 100m CPU, 64Mi Memory; Limits: 500m CPU, 128Mi Memory |
| `probe.liveness.initialDelaySeconds` | Initial delay for liveness probe | `15` |
| `probe.liveness.periodSeconds` | Period for liveness probe | `20` |
| `probe.readiness.initialDelaySeconds` | Initial delay for readiness probe | `5` |
| `probe.readiness.periodSeconds` | Period for readiness probe | `10` |
| `ports.healthProbe` | Health probe port | `8081` |
| `ports.metrics` | Metrics port | `8080` |
| `installCRDs` | Whether to install CRDs as part of the release | `true` |
| `nodeSelector` | Node selectors for scheduling | `{}` |
| `tolerations` | Tolerations for scheduling | `[]` |
| `affinity` | Affinity rules for scheduling | `{}` |
| `leaderElection.enabled` | Enable leader election for controller manager | `true` |
| `leaderElection.id` | Leader election ID | `"aso.yet-another-secrets.io"` |
| `extraEnv` | Extra environment variables | `[]` |
| `podSecurityContext` | Pod security context | `runAsNonRoot: true` |
| `containerSecurityContext` | Container security context | `allowPrivilegeEscalation: false, capabilities.drop: [ALL]` |

## AWS Authentication

### Using Pod Identity (EKS)

Before installing the chart, set up EKS Pod Identity using the AWS CLI:

```bash
aws eks create-pod-identity-association \
    --cluster-name your-eks-cluster \
    --namespace yaso \
    --service-account another-secrets-operator \
    --role-arn arn:aws:iam::123456789012:role/another-secrets-operator-role
```

The chart is designed to work with Pod Identity without requiring any special configuration.

### Using IRSA (older EKS method)

Note: This chart has been simplified to focus on Pod Identity. If you need to use IRSA, you'll need to modify the chart to add annotation support.

### Using Environment Variables

To use environment variables (less secure, not recommended for production):

```yaml
aws:
  region: "us-west-2"  # This sets AWS_REGION env var
```

## Using the Operator

### Create AGenerator

Create a generator for password generation:

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

### Create ASecret

Create a secret with reference to the generator:

```yaml
apiVersion: yet-another-secrets.io/v1alpha1
kind: ASecret
metadata:
  name: my-app-secret
  namespace: yaso
spec:
  targetSecretName: app-credentials
  awsSecretPath: /my-app/secrets
  data:
    username:
      value: admin
    password:
      generatorRef:
        name: password-generator
```

## Uninstalling the Chart

To uninstall/delete the operator deployment:

```bash
helm delete another-secrets-operator
```

Note: This will not delete the CRDs by default. To delete them manually:

```bash
kubectl delete crd asecrets.yet-another-secrets.io
kubectl delete crd agenerators.yet-another-secrets.io
```