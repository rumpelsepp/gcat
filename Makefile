GO ?= go

.PHONY: gcat
gcat:
	$(GO) build $(GOFLAGS) -o $@ ./cmd/$@

.PHONY: test
test:
	$(GO) test ./...

.PHONY: clean
clean:
	$(RM) gcat
