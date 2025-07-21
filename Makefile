monitor_BINARY_NAME = configmap-monitor
VERIFIER_BINARY_NAME = verifier

all: build

.PHONY: build
build: build-monitor build-verifier

.PHONY: build-monitor
build-monitor: fmt vet
	go build -o ./bin/${monitor_BINARY_NAME} ./cmd/monitor

.PHONY: build-verifier
build-verifier: fmt vet
	go build -o ./bin/${VERIFIER_BINARY_NAME} ./cmd/verifier

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...