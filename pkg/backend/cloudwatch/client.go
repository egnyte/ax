package cloudwatch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/egnyte/ax/pkg/backend/common"
)

// TODO
// - Follow
// - Date range support

type CloudwatchClient struct {
	client    *cloudwatchlogs.CloudWatchLogs
	groupName string
}

func attemptParseJSON(str string) map[string]interface{} {
	m := make(map[string]interface{})
	// Find start of JSON blob
	startIdx := strings.Index(str, "{")
	if startIdx == -1 { // If not found, fall back to dumping the whole thing into the "message" field
		m["message"] = str
		return m
	}
	err := json.Unmarshal([]byte(str[startIdx:]), &m)
	if err != nil {
		m["message"] = str
	}
	return m
}

func logEventToMessage(query common.Query, logEvent *cloudwatchlogs.FilteredLogEvent) common.LogMessage {
	message := common.NewLogMessage()
	message.ID = *logEvent.EventId
	message.Timestamp = time.Unix((*logEvent.Timestamp)/1000, (*logEvent.Timestamp)%1000)
	message.Attributes = common.Project(attemptParseJSON(*logEvent.Message), query.SelectFields)
	return message
}

// https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html
func queryToFilterPattern(query common.Query) string {
	filterParts := make([]string, 0)
	for _, filter := range query.Filters {
		filterParts = append(filterParts, fmt.Sprintf("($.%s %s \"%s\")", filter.FieldName, filter.Operator, filter.Value))
	}
	var filterPattern string
	if len(query.Filters) == 0 {
		filterPattern = query.QueryString
	} else {
		filterPattern = fmt.Sprintf("%s { %s }", query.QueryString, strings.Join(filterParts, " && "))
	}

	return strings.TrimSpace(filterPattern)
}

func (client *CloudwatchClient) Query(query common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)

	go func() {
		resp, err := client.client.FilterLogEvents(&cloudwatchlogs.FilterLogEventsInput{
			LogGroupName:  aws.String(client.groupName),
			FilterPattern: aws.String(queryToFilterPattern(query)),
			Limit:         aws.Int64(int64(query.MaxResults)),
		})
		if err != nil {
			panic(err)
		}
		for _, message := range resp.Events {
			resultChan <- logEventToMessage(query, message)
		}
		close(resultChan)
	}()

	return resultChan
}

func (client *CloudwatchClient) ListGroups() ([]string, error) {
	resp, err := client.client.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		return nil, err
	}

	groupNames := make([]string, 0)
	for _, stream := range resp.LogGroups {
		groupNames = append(groupNames, *stream.LogGroupName)
	}

	return groupNames, err
}

func New(accessKey, accessSecretKey, region, groupName string) *CloudwatchClient {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, accessSecretKey, ""),
	})

	if err != nil {
		panic(err)
	}
	client := cloudwatchlogs.New(sess)

	return &CloudwatchClient{
		client:    client,
		groupName: groupName,
	}

}

var _ common.Client = &CloudwatchClient{}
