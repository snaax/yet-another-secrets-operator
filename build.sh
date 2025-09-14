# Set Docker Hub as your registry (with your Docker Hub username)
DOCKER_REGISTRY=docker.io/phuault
DOCKER_IMAGE_NAME=yet-another-secrets-operator
DOCKER_TAG=0.1.0

HELM_REGISTRY=registry-1.docker.io/phuault
HELM_IMAGE_NAME=yet-another-secrets-operator
HELM_VERSION=0.1.0

# Build and push docker image
docker build -t ${DOCKER_REGISTRY}/${DOCKER_IMAGE_NAME}:${DOCKER_TAG} .
docker push ${DOCKER_REGISTRY}/${DOCKER_IMAGE_NAME}:${DOCKER_TAG}

# Build and push helm chart
helm package ./chart/${HELM_IMAGE_NAME}
helm push ${HELM_IMAGE_NAME}-${HELM_VERSION}.tgz oci://${HELM_REGISTRY}
