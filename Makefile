# Makefile

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes
SHELL = /usr/bin/env bash

# Define the controller-gen binary location
CONTROLLER_GEN = $(GOBIN)/controller-gen
# Define the setup-envtest binary location
SETUP_ENVTEST = $(GOBIN)/setup-envtest

# Kubernetes version for envtest
ENVTEST_K8S_VERSION = 1.33.0

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0)

.PHONY: update-helm-crds
update-helm-crds: manifests ## Update Helm chart CRDs while preserving template conditions
	@echo "Updating Helm chart CRDs..."
	@# Create temporary file with the conditional wrapper
	@echo "{{- if .Values.installCRDs }}" > chart/yet-another-secrets-operator/templates/crds.yaml.tmp
	@echo "---" >> chart/yet-another-secrets-operator/templates/crds.yaml.tmp
	@# Append the generated CRDs
	@tail -n +2 config/crd/bases/yet-another-secrets.io_agenerators.yaml >> chart/yet-another-secrets-operator/templates/crds.yaml.tmp
	@echo "---" >> chart/yet-another-secrets-operator/templates/crds.yaml.tmp
	@tail -n +2 config/crd/bases/yet-another-secrets.io_asecrets.yaml >> chart/yet-another-secrets-operator/templates/crds.yaml.tmp
	@echo "{{- end }}" >> chart/yet-another-secrets-operator/templates/crds.yaml.tmp
	@# Replace the original file
	@mv chart/yet-another-secrets-operator/templates/crds.yaml.tmp chart/yet-another-secrets-operator/templates/crds.yaml
	@echo "Helm chart CRDs updated successfully"

# Run tests
.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: test-unit
test-unit: ## Run unit tests only.
	go test ./pkg/... ./api/... -v -coverprofile cover.out

.PHONY: test-integration
test-integration: manifests generate fmt vet envtest ## Run integration tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./tests/... -v

.PHONY: test-coverage
test-coverage: test-unit
	go tool cover -html=cover.out -o coverage.html

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
ENVTEST_K8S_VERSION = 1.31.0

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
