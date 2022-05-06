GO ?= go

.PHONY: gcat
gcat:
	$(GO) build $(GOFLAGS) -o $@ ./cmd/$@

.PHONY: update
update:
	$(GO) get -u ./cmd/gcat
	$(GO) mod tidy

.PHONY: test
test:
	$(GO) test ./...

.PHONY: clean
clean:
	$(RM) gcat
