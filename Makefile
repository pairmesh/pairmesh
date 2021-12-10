GOVER := $(shell go version)

GOOS    := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
GOARCH  := $(if $(GOARCH),$(GOARCH),amd64)
GOENV   := GO111MODULE=on CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO      := $(GOENV) go
GOBUILD := $(GO) build $(BUILD_FLAG)
GOTEST  := GO111MODULE=on CGO_ENABLED=1 $(GO) test -p 3
SHELL   := /usr/bin/env bash

COMMIT    := $(shell git describe --no-match --always --dirty)
BRANCH    := $(shell git rev-parse --abbrev-ref HEAD)
BUILDTIME := $(shell date '+%Y-%m-%d %T %z')

REPO := github.com/pairmesh/pairmesh
LDFLAGS := -w -s
LDFLAGS += -X "$(REPO)/version.GitHash=$(COMMIT)"
LDFLAGS += -X "$(REPO)/version.GitBranch=$(BRANCH)"
LDFLAGS += $(EXTRA_LDFLAGS)

FILES     := $$(find . -name "*.go")

FAILPOINT_ENABLE  := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl enable)
FAILPOINT_DISABLE := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl disable)

default: fmt pairmesh pairportal pairrelay

fmt:
	@echo "gofmt (simplify)"
	@gofmt -s -l -w $(FILES) 2>&1

proto:
	@cd message/protos; \
    protoc --go_out=. *.proto; \
    protoc --go-grpc_out=. *.proto

pairmesh:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairmesh ./cmd/pairmesh

pairportal:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairportal ./cmd/pairportal

pairrelay:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairrelay ./cmd/pairrelay

queryset:
	go run ./tools/qs/main.go -in ./cmd/pairportal/db/models/models.go -out ./cmd/pairportal/db/models/autogen_query.go

.PHONY: build package
