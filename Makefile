.PHONY: run build test test-race templ fmt vet tidy clean

APP=./cmd/server
BIN=bin/ledger-server

run: templ
	go run $(APP)

build: templ
	go build -o $(BIN) $(APP)

test:
	go test ./...

test-race:
	go test -race ./...

templ:
	templ generate

fmt:
	go fmt ./...
	templ fmt .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
	rm -f data/*.db data/*.db-wal data/*.db-shm
