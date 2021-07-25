GO ?= go

.PHONY: gcat
gcat:
	$(GO) build $(GOFLAGS) -o $@ ./bin/$@
