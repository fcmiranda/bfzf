.PHONY: build install install-linux uninstall-linux clean

BINARY := bfzf
CMD    := ./cmd/bfzf
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

## build: compile and place the binary in the project root (run with ./bfzf)
build:
	go build -o $(BINARY) $(CMD)

## install: install into GOPATH/bin (available system-wide as `bfzf`)
install:
	go install $(CMD)

## install-linux: install binary into ~/.local/bin (Linux-friendly global install)
install-linux: build
	mkdir -p "$(BINDIR)"
	install -m 0755 "$(BINARY)" "$(BINDIR)/$(BINARY)"
	@printf '\nInstalled %s to %s\n' "$(BINARY)" "$(BINDIR)"
	@printf 'Ensure %s is on PATH, e.g. add to ~/.bashrc or ~/.zshrc:\n' "$(BINDIR)"
	@printf '  export PATH="%s:$$PATH"\n\n' "$(BINDIR)"

## uninstall-linux: remove binary from ~/.local/bin
uninstall-linux:
	rm -f "$(BINDIR)/$(BINARY)"

## clean: remove the local binary
clean:
	rm -f $(BINARY)
