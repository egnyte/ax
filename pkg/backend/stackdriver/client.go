package stackdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/api/option"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/egnyte/ax/pkg/backend/common"
	"google.golang.org/api/iterator"
)

type StackdriverClient struct {
	stackdriverClient *logadmin.Client
	projectName       string
	logName           string
}

type payLoadValue struct {
	Fields map[string]payLoadEntry
}

type payLoadEntry struct {
	Kind struct {
		StringValue *string
		NumberValue *int64
		BoolValue   *bool
		ListValue   *struct {
			Values []payLoadEntry
		}
		StructValue *payLoadValue
	}
}

func payLoadValueToJSONValue(plVal payLoadValue) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range plVal.Fields {
		m[k] = payLoadEntryToJSONValue(v)
	}
	return m
}

func payLoadEntryToJSONValue(plEntry payLoadEntry) interface{} {
	kind := plEntry.Kind
	if kind.StringValue != nil {
		return *plEntry.Kind.StringValue
	} else if kind.NumberValue != nil {
		return *kind.NumberValue
	} else if kind.BoolValue != nil {
		return *kind.BoolValue
	} else if kind.ListValue != nil {
		list := make([]interface{}, len((*kind.ListValue).Values))
		for idx, val := range (*kind.ListValue).Values {
			list[idx] = payLoadEntryToJSONValue(val)
		}
		return list
	} else if kind.StructValue != nil {
		return payLoadValueToJSONValue(*kind.StructValue)
	} else {
		return nil
	}
}

func payloadToAttributes(buf []byte) map[string]interface{} {
	var plValue payLoadValue
	if err := json.Unmarshal(buf, &plValue); err != nil {
		fmt.Printf("Could not unmarshall value: %s", string(buf))
		return nil
	}
	return payLoadValueToJSONValue(plValue)
}

func entryToLogMessage(entry *logging.Entry) common.LogMessage {
	message := common.NewLogMessage()
	message.Timestamp = entry.Timestamp
	message.ID = entry.InsertID
	switch v := entry.Payload.(type) {
	case string:
		message.Attributes["message"] = v
	case map[string]interface{}:
		message.Attributes = v
	default:
		buf, err := json.Marshal(entry.Payload)
		if err != nil {
			fmt.Printf("Could not marshall value: %v of type %v", entry.Payload, reflect.TypeOf(entry))
			break
		}
		// fmt.Println("JSON: ", string(buf))
		message.Attributes = payloadToAttributes(buf)
	}
	return message
}

func (client *StackdriverClient) Query(ctx context.Context, query common.Query) <-chan common.LogMessage {
	it := client.stackdriverClient.Entries(ctx, logadmin.Filter(fmt.Sprintf(`logName = "projects/%s/logs/%s"`, client.projectName, client.logName)))
	resultChan := make(chan common.LogMessage)

	go func() {
		entry, err := it.Next()
		if err != nil && err != iterator.Done {
			fmt.Printf("Error retrieving logs: %s\n", err)
			close(resultChan)
		}
		for err != iterator.Done {
			resultChan <- entryToLogMessage(entry)
			entry, err = it.Next()
		}
		close(resultChan)
	}()

	return resultChan
}

func New(credentialsFile, projectName, logName string) *StackdriverClient {
	client, err := logadmin.NewClient(context.Background(), projectName, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		fmt.Printf("Error creating stack driver client: %v\n", err)
		return nil
	}
	return &StackdriverClient{
		stackdriverClient: client,
		projectName:       projectName,
		logName:           logName,
	}
}

func (client *StackdriverClient) ListLogs(ctx context.Context) ([]string, error) {
	logNames := make([]string, 0, 10)
	it := client.stackdriverClient.Logs(ctx)
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

var _ common.Client = &StackdriverClient{}
