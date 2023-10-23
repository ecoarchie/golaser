build:
	@go build -o bin/golaser

run: build
	@./bin/golaser

test:
	@go test -v ./...