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

# go-get-tool will 'go install' any package with specified version
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: setup-envtest
setup-envtest: ## Download setup-envtest locally if necessary.
	$(call go-get-tool,$(SETUP_ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

.PHONY: envtest
envtest: setup-envtest ## Download and set up envtest binaries
	@mkdir -p testbin
	@echo "Setting up envtest binaries in testbin..."
	@$(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir testbin

.PHONY: test
test: generate manifests envtest ## Run tests
	KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use -p path $(ENVTEST_K8S_VERSION))" go test ./... -v

.PHONY: test-coverage
test-coverage: generate manifests envtest ## Run tests with coverage
	KUBEBUILDER_ASSETS="$(shell $(SETUP_ENVTEST) use -p path $(ENVTEST_K8S_VERSION))" go test ./... -v -coverprofile cover.out

.PHONY: update-test-env
update-test-env: ## Update the tests/suite_test.go to use the testbin directory
	@echo "Updating test environment setup..."
	@if grep -q "BinaryAssetsDirectory" tests/suite_test.go; then \
		echo "BinaryAssetsDirectory already configured"; \
	else \
		sed -i -e 's/testEnv = \&envtest.Environment{/testEnv = \&envtest.Environment{\n\t\tBinaryAssetsDirectory: filepath.Join("..", "testbin"),/g' tests/suite_test.go; \
	fi
