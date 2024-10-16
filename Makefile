ARTIFACTS ?= demo
COVERAGE_DIR=$(BUILD_DIR)/coverage
GO_COVER_FILE=$(COVERAGE_DIR)/go-coverage.txt

include build/base.mk
include build/k8s.mk

main:
	CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o $(BUILD_DIR)/kompile ./cmd/.
	cd demo && CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o $(BUILD_DIR)/demo main.go

lint:
	golangci-lint run

test:
	mkdir -p $(COVERAGE_DIR)
	go test -v -coverprofile=$(GO_COVER_FILE) ./...

cover:
	go tool cover -func=$(GO_COVER_FILE)

pre-k8s::
	sed -e "s|PLACEHOLDER|$(shell cat $(BUILD_DIR)/demo-image)|g" demo/k8s/demo.yml.tmpl > $(K8S_MANIFESTS_DIR)/demo.yml
