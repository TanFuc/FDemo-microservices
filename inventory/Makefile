.PHONY: build run test clean docker-up docker-down tidy

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./internal/usecase/...

test-stress:
	go test -v -run TestConcurrencyStressTest ./internal/usecase/...

clean:
	rm -rf bin/

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

tidy:
	go mod tidy

all: tidy build
