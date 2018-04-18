package file

import (
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/backend/stream"
)

type FileClient struct {
	filename string
}

func openReader(filename string) (io.Reader, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(filename, ".gz") {
		return gzip.NewReader(r)
	}
	if strings.HasSuffix(filename, ".bz2") {
		bzipReader := bzip2.NewReader(r)
		return bzipReader, nil
	}
	return r, nil
}

func (client *FileClient) Query(ctx context.Context, query common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	reader, err := openReader(client.filename)
	if err != nil {
		fmt.Printf("Could not open file %s for reading: %s", client.filename, err.Error())
		os.Exit(1)
	}
	streamReader := stream.New(reader)
	go func() {
		for message := range streamReader.Query(ctx, query) {
			resultChan <- message
		}
		close(resultChan)
	}()
	return resultChan
}

func New(filename string) *FileClient {
	return &FileClient{filename}
}

var _ common.Client = &FileClient{}
