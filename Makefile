APP_NAME=bar-extract
VERSION?=v0.1.0
LDFLAGS=-X main.version=$(VERSION)
GOFILES=$(shell find . -name '*.go' -not -path './.devcontainer/*')

.PHONY: build test clean build-all macos-app

build: $(GOFILES)
	GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME) ./cmd/bar-extract

build-all: ## Cross-compile common targets
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-linux-amd64 ./cmd/bar-extract
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-windows-amd64.exe ./cmd/bar-extract
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-darwin-arm64 ./cmd/bar-extract
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(APP_NAME)-darwin-amd64 ./cmd/bar-extract

macos-app: build ## Create a minimal .app bundle (macOS only)
	bash scripts/build_macos_app.sh $(VERSION)

test:
	go test ./...

clean:
	rm -rf dist *.log *.out
