ifeq ($(OS), Windows_NT)
	VERSION := $(shell git describe --exact-match --tags 2>nil)
	HOME := $(HOMEPATH)
	CGO_ENABLED ?= 0
	export CGO_ENABLED
else
	VERSION := $(shell git describe --exact-match --tags 2>/dev/null)
endif

PREFIX := /usr/local
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git rev-parse --short HEAD)
GOFILES ?= $(shell git ls-files '*.go')
GOFMT ?= $(shell gofmt -l -s $(GOFILES))
BUILDFLAGS ?=

ifdef GOBIN
PATH := $(GOBIN):$(PATH)
else
PATH := $(subst :,/bin:,$(shell go env GOPATH))/bin:$(PATH)
endif

LDFLAGS := $(LDFLAGS) -X main.commit=$(COMMIT) -X main.branch=$(BRANCH)
ifdef VERSION
	LDFLAGS += -X main.version=$(VERSION)
endif

.PHONY: all
all:
	@$(MAKE) --no-print-directory deps
	@$(MAKE) --no-print-directory base64url

.PHONY: deps
deps:
	go mod download

.PHONY: base64url
base64url:
	go build -ldflags "$(LDFLAGS)" ./cmd/base64url/base64url.go

.PHONY: go-install
go-install:
	go install -ldflags "-w -s $(LDFLAGS)" ./cmd/base64url/base64url.go

.PHONY: install
install: base64url
	mkdir -p $(DESTDIR)$(PREFIX)/bin/
	cp base64url $(DESTDIR)$(PREFIX)/bin/

.PHONY: uninstall
uninstall: 
	@echo "[CONFIRM]"
	@rm -i $(DESTDIR)$(PREFIX)/bin/base64url


.PHONY: test
test:
	go test -short `go list`

.PHONY: fmt
fmt:
	@gofmt -s -w $(GOFILES)

.PHONY: fmtcheck
fmtcheck:
	@if [ ! -z "$(GOFMT)" ]; then \
		echo "[ERROR] gofmt has found errors in the following files:"  ; \
		echo "$(GOFMT)" ; \
		echo "" ;\
		echo "Run make fmt to fix them." ; \
		exit 1 ;\
	fi

.PHONY: vet
vet:
	@echo 'go vet $$(go list ./... )'
	@go vet $$(go list ./... ) ; if [ $$? -ne 0 ]; then \
		echo ""; \
		echo "go vet has found suspicious constructs. Please remediate any reported errors"; \
		echo "to fix them before submitting code for review."; \
		exit 1; \
	fi

.PHONY: tidy
tidy:
	go mod verify
	go mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "please run go mod tidy and check in changes"; \
		exit 1; \
	fi

.PHONY: check
check: fmtcheck vet
	@$(MAKE) --no-print-directory tidy

.PHONY: test-all
test-all: fmtcheck vet
	go test `go list`

.PHONY: clean
clean:
	rm -f base64url
	rm -f base64url.exe

.PHONY: static
static:
	@echo "Building static linux binary..."
	@CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	go build -ldflags "$(LDFLAGS) -s -w" ./cmd/base64url/base64url.go