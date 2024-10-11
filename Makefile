ARTIFACTS ?= kompile
COVERAGE_DIR=$(BUILD_DIR)/coverage
GO_COVER_FILE=$(COVERAGE_DIR)/go-coverage.txt

include build/base.mk
include build/k8s.mk

main: lint
	CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o $(BUILD_DIR)/kompile ./cmd/.

lint:
	golangci-lint run

test:
	mkdir -p $(COVERAGE_DIR)
	go test -v -coverprofile=$(GO_COVER_FILE) ./...

cover:
	go tool cover -func=$(GO_COVER_FILE)
