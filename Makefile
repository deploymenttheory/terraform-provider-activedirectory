GO ?= go

.PHONY: generate-docs check-docs

generate-docs:
	$(GO) run ./internal/tools/docscheck
	$(GO) generate ./...

check-docs: generate-docs
	git diff --exit-code -- docs templates examples
	test -z "$$(git ls-files --others --exclude-standard -- docs templates examples)"
