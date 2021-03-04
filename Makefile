GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=khcheck-external-secrets

.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/check/
