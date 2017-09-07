package kibana

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/zefhemel/ax/pkg/backend/common"

	"os"

	"time"
)

type JsonObject map[string]interface{}
type JsonList []interface{}

type QueryResult struct {
	Responses []struct {
		Hits struct {
			Hits []Hit `json:"hits"`
		} `json:"hits"`
	} `json:"responses"`
}

type Hit struct {
	ID     string     `json:"_id"`
	Source JsonObject `json:"_source"`
}

type hitsByAscDate []Hit

func (a hitsByAscDate) Len() int      { return len(a) }
func (a hitsByAscDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a hitsByAscDate) Less(i, j int) bool {
	return a[i].Source["@timestamp"].(string) < a[j].Source["@timestamp"].(string)
}

func (client *Client) queryMessages(subIndex string, query common.Query) ([]Hit, error) {
	queryString := fmt.Sprintf("\"%s\"", query.QueryString) // TODO: Handle quotes properly
	if query.QueryString == "" {
		queryString = "*"
	}
	mustFilters := JsonList{
		JsonObject{
			"query_string": JsonObject{
				"analyze_wildcard": true,
				"query":            queryString,
			},
		},
	}

	if query.After != nil || query.Before != nil {
		rangeObj := JsonObject{
			"range": JsonObject{
				"@timestamp": JsonObject{
					"format": "epoch_millis",
				},
			},
		}
		if query.After != nil {
			rangeObj["range"].(JsonObject)["@timestamp"].(JsonObject)["gt"] = unixMillis(*query.After)
		}
		if query.Before != nil {
			rangeObj["range"].(JsonObject)["@timestamp"].(JsonObject)["lt"] = unixMillis(*query.Before)
		}
		mustFilters = append(mustFilters, rangeObj)
	}
	mustNotFilters := JsonList{}
	for _, filter := range query.Filters {
		m := JsonObject{}
		switch filter.Operator {
		case "=":
			m[filter.FieldName] = JsonObject{
				"query": filter.Value,
				"type":  "phrase",
			}
			mustFilters = append(mustFilters, JsonObject{
				"match": m,
			})
		case "!=":
			m[filter.FieldName] = JsonObject{
				"query": filter.Value,
				"type":  "phrase",
			}
			mustNotFilters = append(mustNotFilters, JsonObject{
				"match": m,
			})
		}
	}
	body, err := createMultiSearch(
		JsonObject{
			"index":              JsonList{subIndex},
			"ignore_unavailable": true,
		},
		JsonObject{
			"size": query.MaxResults,
			"sort": JsonList{
				JsonObject{
					"@timestamp": JsonObject{
						"order":         "desc",
						"unmapped_type": "boolean",
					},
				},
			},
			"query": JsonObject{
				"bool": JsonObject{
					"must":     mustFilters,
					"must_not": mustNotFilters,
				},
			},
		})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/elasticsearch/_msearch", client.URL), body)
	if err != nil {
		return nil, err
	}
	client.addHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var data QueryResult
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	hits := data.Responses[0].Hits.Hits
	sort.Sort(hitsByAscDate(hits))
	return hits, nil
}

func (client *Client) queryFollow(q common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	go func() {
		var after *time.Time
		retries := 0
		for {
			q.After = after
			allMessages, err := client.querySubIndex(client.Index, q)
			if err != nil {
				retries++
				if retries < 10 {
					fmt.Fprintf(os.Stderr, "Could not connect to Kibana: %v retrying in 5s\n", err)
					time.Sleep(5 * time.Second)
					continue
				} else {
					fmt.Fprintf(os.Stderr, "Could not connect to Kibana: %v\nExceeded total number of retries, exiting.\n", err)
					os.Exit(1)
				}
			}
			// Request succesful, so reset retry count
			retries = 0
			for _, message := range allMessages {
				resultChan <- message
				after = &message.Timestamp
			}
			if after == nil {
				fmt.Fprintf(os.Stderr, "Could determine latest hit, defaulting to now")
				afterDate := time.Now()
				after = &afterDate
			}
			time.Sleep(5 * time.Second)
		}
	}()
	return resultChan
}

func (client *Client) Query(q common.Query) <-chan common.LogMessage {
	resultChan := make(chan common.LogMessage)
	if q.Before == nil {
		before := time.Now().Add(12 * time.Hour)
		q.Before = &before // Limit sanity
	}
	if q.Follow {
		return client.queryFollow(q)
	}
	go func() {
		printedResultsCount := 0
		fmt.Fprintf(os.Stderr, "Querying index %s\n", client.Index)
		allMessages, err := client.querySubIndex(client.Index, q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not connect to Kibana: %v", err)
			os.Exit(2)
		}
		for _, message := range allMessages {
			resultChan <- message
			printedResultsCount++
			if printedResultsCount >= q.MaxResults {
				break
			}
		}
		close(resultChan)
	}()

	return resultChan
}

func (client *Client) querySubIndex(subIndex string, q common.Query) ([]common.LogMessage, error) {
	hits, err := client.queryMessages(subIndex, q)
	if err != nil {
		return nil, err
	}

	allMessages := make([]common.LogMessage, 0, 200)
	for _, hit := range hits {
		//var ts time.Time
		attributes := hit.Source
		ts, err := time.Parse(time.RFC3339, attributes["@timestamp"].(string))
		if err != nil {
			return nil, err
		}
		delete(attributes, "@timestamp")
		message := common.FlattenLogMessage(common.LogMessage{
			ID:         hit.ID,
			Timestamp:  ts,
			Attributes: attributes,
		})
		message.Attributes = common.Project(message.Attributes, q.SelectFields)
		allMessages = append(allMessages, message)
	}
	return allMessages, nil
}
