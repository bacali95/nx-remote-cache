.PHONY: build test run docker

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -race -cover

run: build
	CACHE_READ_TOKENS=dev-read-token CACHE_WRITE_TOKENS=dev-write-token ./bin/server

docker:
	docker build -t nx-remote-cache:local .
