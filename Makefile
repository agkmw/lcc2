BINARY := lcc2
PKG    := ./cmd/lcc2
BIN    := bin

.PHONY: all run build install check vet test fmt cover clean

all: check

run:
	go run $(PKG)

build:
	go build -trimpath -o $(BIN)/$(BINARY) $(PKG)

install:
	go install $(PKG)

check:
	./scripts/check.sh

vet:
	go vet ./...

test:
	go test ./...

fmt:
	go fmt ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -rf $(BIN) coverage.out
