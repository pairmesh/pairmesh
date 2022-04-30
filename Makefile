GOVER := $(shell go version)

GOOS    := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
GOARCH  := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
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
LINT_DIRS := $$(go list ./... | grep -vE "wintun|tools|systray|macos")

FAILPOINT_ENABLE  := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl enable)
FAILPOINT_DISABLE := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl disable)

default: fmt pairmesh pairportal pairrelay pairbench

pairbench:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairbench ./cmd/pairbench

pairmesh:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairmesh ./cmd/pairmesh

pairportal:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairportal ./cmd/pairportal

pairrelay:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/pairrelay ./cmd/pairrelay

test:
	$(GOTEST) `go list ./... | grep -v tools | grep -v systray`

integration_test:

# Check
# Lint tools
check: fmt vet check-static lint

clean:
	rm -rf ./bin

.PHONY: build package

include build/*.mk
