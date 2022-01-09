GO ?= go

.PHONY: gcat
gcat:
	$(GO) build $(GOFLAGS) -o $@ ./bin/$@

.PHONY: clean
clean:
	$(RM) gcat
