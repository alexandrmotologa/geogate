.PHONY: all build test bench web-build run clean

BINARY_NAME=geogate

all: web-build build test

web-build:
	cd web && npm install && npm run build

build:
	go build -ldflags="-s -w" -o bin/$(BINARY_NAME) ./cmd/geogate

test:
	go test -v ./internal/... ./test/...

bench:
	go test -bench=. -benchmem ./internal/...

run: build
	./bin/$(BINARY_NAME) serve -upstream http://localhost:3000 -port 8080

clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -f geogate geogate.exe
