# Install mmdc https://github.com/mermaid-js/mermaid-cli
MMD_CMD = mmdc -t neutral
COMPOSITE_CONTROLLER_DIR = controller/composite/v1

BIN_PATH = $(shell pwd)/bin
# Since all the external tools are placed in BIN_PATH, append $PATH with
# BIN_PATH.
export PATH := $(BIN_PATH):$(PATH)

# KUBEBUILDER_ASSETS path is set as environment variable when running envtest.
ENVTEST_BIN_VERSION = 1.19.2
KUBEBUILDER_ASSETS = $(shell $(SETUP_ENVTEST) use -i -p path $(ENVTEST_BIN_VERSION))

generate: mockgen
	go generate ./...

test: generate setup-envtest
	KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS) \
		go test -v -timeout 300s -race ./... -count=1 \
		-coverprofile cover.out

update-diagrams:
	$(MMD_CMD) -i $(COMPOSITE_CONTROLLER_DIR)/docs/create.mmd -o $(COMPOSITE_CONTROLLER_DIR)/docs/create.svg
	$(MMD_CMD) -i $(COMPOSITE_CONTROLLER_DIR)/docs/update.mmd -o $(COMPOSITE_CONTROLLER_DIR)/docs/update.svg
	$(MMD_CMD) -i $(COMPOSITE_CONTROLLER_DIR)/docs/delete.mmd -o $(COMPOSITE_CONTROLLER_DIR)/docs/delete.svg

clean:
	rm -rf $(BIN_PATH)

MOCKGEN = $(shell pwd)/bin/mockgen
mockgen:
	$(call go-get-tool,$(MOCKGEN),github.com/golang/mock/mockgen@latest)

SETUP_ENVTEST = $(shell pwd)/bin/setup-envtest
setup-envtest:
	$(call go-get-tool,$(SETUP_ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)
	$(SETUP_ENVTEST) use $(ENVTEST_BIN_VERSION)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
