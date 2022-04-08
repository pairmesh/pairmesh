fmt:
	@echo "gofmt (simplify)"
	@gofmt -s -l -w $(FILES) 2>&1

lint: tools/bin/revive
	@tools/bin/revive -formatter friendly -config tools/check/revive.toml $(LINT_DIRS)

vet:
	$(GO) vet $(LINT_DIRS)

check-static: tools/bin/golangci-lint
	tools/bin/golangci-lint run --timeout 5m ./...

tools/bin/revive: tools/check/go.mod
	cd tools/check; \
	$(GO) build -o ../bin/revive github.com/mgechev/revive

tools/bin/golangci-lint: tools/check/go.mod
	cd tools/check; \
	$(GO) build -o ../bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint