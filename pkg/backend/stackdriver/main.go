package main

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"google.golang.org/api/iterator"
)

func writeLogs() {
	ctx := context.Background()
	client, err := logging.NewClient(ctx, "turbo-service")
	if err != nil {
		panic(err)
	}
	lg := client.Logger("my-test-log")

	// Add entry to log buffer
	err = lg.LogSync(ctx, logging.Entry{Payload: json.RawMessage([]byte(`{"name": "Zef", "age": 34}`))})
	if err != nil {
		panic(err)
	}
	client.Close()
}

func listLogs(ctx context.Context, client *logadmin.Client) ([]string, error) {
	logNames := make([]string, 0, 10)
	it := client.Logs(ctx)
	s, err := it.Next()
	if err != nil && err != iterator.Done {
		return nil, err
	}
	for err != iterator.Done {
		logNames = append(logNames, s)
		s, err = it.Next()
		if err != nil && err != iterator.Done {
			return nil, err
		}
	}
	return logNames, nil
}

func getLogs(ctx context.Context, client *logadmin.Client, logName string) {
	it := client.Entries(ctx, logadmin.Filter(fmt.Sprintf(`logName = "projects/turbo-service/logs/%s"`, logName)))
	entry, err := it.Next()
	if err != nil && err != iterator.Done {
		panic(err)
	}
	for err != iterator.Done {
		fmt.Println(entry.Timestamp, entry.Payload)
		entry, err = it.Next()
	}
}

func main() {
	ctx := context.Background()
	client, err := logadmin.NewClient(ctx, "turbo-service")

	if err != nil {
		panic(err)
	}

	// names, err := listLogs(ctx, client)
	// fmt.Println(names, err)
	getLogs(ctx, client, "my-test-log")
}
