# よく使うコマンドを短く呼び出すためのMakefileです。

.PHONY: run tidy test

run:
	go run ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...
