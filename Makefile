all:
	go install github.com/egnyte/ax/cmd/ax

test:
	go test ./cmd/... ./pkg/...

deps:
	go get ./cmd/... ./pkg/...
