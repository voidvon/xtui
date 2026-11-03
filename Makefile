.PHONY: all build build-linux clean run

BINARY=xtui

all: build

build:
	go build -ldflags="-s -w" -trimpath -o $(BINARY) .

# Statically linked Linux AMD64 (VPS)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o $(BINARY)-linux-amd64 .

# Statically linked Linux ARM64 (Oracle Cloud / Raspberry Pi)
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o $(BINARY)-linux-arm64 .

run:
	go run main.go

clean:
	rm -f $(BINARY) $(BINARY)-linux-*
