all:
	dep ensure
	go install github.com/egnyte/ax/cmd/ax

test:
	go test ./cmd/... ./pkg/...

deps:
	dep ensure

release:
	rm -rf dist bin
	goreleaser
