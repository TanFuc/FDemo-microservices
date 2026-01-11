.PHONY: build run test clean docker-build docker-up docker-down deps

build:
	go build -o bin/api-gateway ./cmd/gateway

run:
	go run ./cmd/gateway -config config.yaml

test:
	go test -v ./...

clean:
	rm -rf bin/

deps:
	go mod download
	go mod tidy

docker-build:
	docker build -t api-gateway:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

lint:
	golangci-lint run ./...
