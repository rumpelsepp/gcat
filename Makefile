GO ?= go

.PHONY: gcat
gcat:
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $@ ./cmd/$@

.PHONY: update
update:
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: test
test:
	$(GO) test ./...

.PHONY: clean
clean:
	$(RM) gcat
