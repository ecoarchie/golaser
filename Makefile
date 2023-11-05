build:
	@go build -o bin/golaser

run: build
	@./bin/golaser

test:
	@go test -v ./...

buildexe:
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build