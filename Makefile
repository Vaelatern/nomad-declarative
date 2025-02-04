.PHONY: build clean test

build:
	go build ./cmd/nomad-declarative

clean:
	rm -f nomad-declarative

test:
	go test ./...
