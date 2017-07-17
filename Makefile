all:
	go install github.com/zefhemel/ax/cmd/ax

test:
	go test ./cmd/... ./pkg/...

deps:
	go get ./cmd/... ./pkg/...
