VERSION ?= dev
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BINARY_NAME := sqlmon
TARGETS := windows/amd64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

test:
	@go test -v -cover -short ./...
run:
	@go run ./cmd --conn "sqlserver://sa:P%40ssW0rd@localhost"

tidy:
	@go mod tidy

build: $(TARGETS)
	
$(TARGETS):
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval EXT := $(if $(filter windows,$(GOOS)),.exe,))
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" -o ./bin/$(BINARY_NAME).$(GOOS).$(GOARCH)$(EXT) ./cmd

clean:
	@rm -rf bin

listdepupdates:
	@go list -m -u all

updatedeps:
	@go get -u

.PHONY: run tidy
