.PHONY: clean build test

clean:
	rm -rf vul-webhook

build:
	go build -o vul-webhook .

test:
	go test -race -v ./...
