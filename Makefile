.PHONY: build install clean

BINARY := bfzf
CMD    := ./cmd/bfzf

## build: compile and place the binary in the project root (run with ./bfzf)
build:
	go build -o $(BINARY) $(CMD)

## install: install into GOPATH/bin (available system-wide as `bfzf`)
install:
	go install $(CMD)

## clean: remove the local binary
clean:
	rm -f $(BINARY)
