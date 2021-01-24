# Install mmdc https://github.com/mermaid-js/mermaid-cli
MMD_CMD = mmdc -t neutral
COMPOSITE_CONTROLLER_DIR = controller/composite/v1

generate: mockgen
	go generate ./...

mockgen:
	GO111MODULE=on go get -v github.com/golang/mock/mockgen@latest

test: generate
	go test -v -race ./... -count=1 -coverprofile cover.out

update-diagrams:
	$(MMD_CMD) -i $(COMPOSITE_CONTROLLER_DIR)/docs/create.mmd -o $(COMPOSITE_CONTROLLER_DIR)/docs/create.svg
	$(MMD_CMD) -i $(COMPOSITE_CONTROLLER_DIR)/docs/update.mmd -o $(COMPOSITE_CONTROLLER_DIR)/docs/update.svg
	$(MMD_CMD) -i $(COMPOSITE_CONTROLLER_DIR)/docs/delete.mmd -o $(COMPOSITE_CONTROLLER_DIR)/docs/delete.svg
