package stackdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"google.golang.org/api/option"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/egnyte/ax/pkg/backend/common"
	"google.golang.org/api/iterator"
)

const QueryLogTimeout = 20 * time.Second

type StackdriverClient struct {
	stackdriverClient *logadmin.Client
	projectName       string
	logName           string
}

// This is some crazy-ass structure in which the stackdriver APIs
// return its JSON values that we have to decode
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
		message.Attributes = payloadToAttributes(buf)
	}
	return message
}

func queryToFilter(query common.Query, projectName string, logName string) string {
	pieces := []string{fmt.Sprintf(`logName = "projects/%s/logs/%s"`, projectName, logName)}
	if query.QueryString != "" {
		pieces = append(pieces, fmt.Sprintf(`"%s"`, query.QueryString))
	}
	for _, filter := range query.Filters {
		pieces = append(pieces, fmt.Sprintf(`jsonPayload.%s %s "%s"`, filter.FieldName, filter.Operator, filter.Value))
	}
	if query.After != nil {
		pieces = append(pieces, fmt.Sprintf(`timestamp > "%s"`, (*query.After).Format(time.RFC3339)))
	}
	if query.Before != nil {
		pieces = append(pieces, fmt.Sprintf(`timestamp < "%s"`, (*query.Before).Format(time.RFC3339)))
	}
	return strings.Join(pieces, " AND ")
}

func (client *StackdriverClient) readLogBatch(ctx context.Context, query common.Query) ([]common.LogMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryLogTimeout)
	defer cancel()
	it := client.stackdriverClient.Entries(ctx, logadmin.Filter(queryToFilter(query, client.projectName, client.logName)))
	messages := make([]common.LogMessage, 0, 20)
	entry, err := it.Next()
	// Somehow, if no results can be found, it.Next() just runs forever, hence the adding a timeout to the context
	if ctx.Err() == context.DeadlineExceeded {
		return messages, ctx.Err()
	}
	if err != nil && err != iterator.Done {
		return nil, err
	}
	resultCounter := 1
	for err != iterator.Done && resultCounter <= query.MaxResults {
		messages = append(messages, entryToLogMessage(entry))
		entry, err = it.Next()
		resultCounter++
	}
	return messages, nil
}

func (client *StackdriverClient) Query(ctx context.Context, query common.Query) <-chan common.LogMessage {
	if query.Follow {
		return common.ReQueryFollow(ctx, func() ([]common.LogMessage, error) {
			return client.readLogBatch(ctx, query)
		})
	}
	resultChan := make(chan common.LogMessage)

	go func() {
		messages, err := client.readLogBatch(ctx, query)
		if err != nil {
			fmt.Printf("Error while fetching logs: %s\n", err)
			close(resultChan)
			return
		}
		for _, message := range messages {
			resultChan <- message
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

func (client *StackdriverClient) ListLogs() ([]string, error) {
	logNames := make([]string, 0, 10)
	it := client.stackdriverClient.Logs(context.Background())
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
