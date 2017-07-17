package common

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Client interface {
	Query(query Query) <-chan LogMessage
}

type QueryFilter struct {
	FieldName string
	Value     string
}

type Query struct {
	QueryString  string
	After        *time.Time
	Before       *time.Time
	SelectFields []string
	Filters      []QueryFilter
	MaxResults   int
	// QueryAsc     bool
	// ResultsDesy  bool
	Follow bool
}

type LogMessage struct {
	Timestamp  time.Time
	Message    string
	Attributes map[string]interface{}
}

func NewLogMessage() LogMessage {
	return LogMessage{
		Attributes: make(map[string]interface{}),
	}
}

func MustJsonEncode(obj interface{}) string {
	buf, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

func MustJsonDecode(jsonString string, dst interface{}) {
	decoder := json.NewDecoder(strings.NewReader(jsonString))
	err := decoder.Decode(dst)
	if err != nil {
		panic(err)
	}
}

func FlattenAttributes(m, into map[string]interface{}, prefix string) {
	for k, gv := range m {
		switch v := gv.(type) {
		case map[string]interface{}:
			FlattenAttributes(v, into, fmt.Sprintf("%s%s.", prefix, k))
		default:
			into[fmt.Sprintf("%s%s", prefix, k)] = gv
		}
	}
}

func FlattenLogMessage(message LogMessage) LogMessage {
	newMessage := NewLogMessage()
	newMessage.Timestamp = message.Timestamp
	newMessage.Message = message.Message
	FlattenAttributes(message.Attributes, newMessage.Attributes, "")
	return newMessage
}

func Project(m map[string]interface{}, fields []string) map[string]interface{} {
	if len(fields) == 0 {
		return m
	}
	projected := make(map[string]interface{})
	for _, field := range fields {
		if val, ok := m[field]; ok {
			projected[field] = val
		}
	}
	return projected
}

func (f QueryFilter) Matches(m LogMessage) bool {
	val, ok := m.Attributes[f.FieldName]
	return ok && f.Value == fmt.Sprintf("%v", val)
}

func matchesPhrase(s, phrase string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(phrase))
}

func MatchesQuery(m LogMessage, q Query) bool {
	matchFound := matchesPhrase(m.Message, q.QueryString)
	if q.QueryString != "" {
		for _, v := range m.Attributes {
			if vs, ok := v.(string); ok {
				if matchesPhrase(vs, q.QueryString) {
					matchFound = true
				}
			}
		}
	}
	if q.Before != nil {
		if m.Timestamp.After(*q.Before) {
			return false
		}
	}
	if q.After != nil {
		if m.Timestamp.Before(*q.After) {
			return false
		}
	}
	for _, f := range q.Filters {
		if !f.Matches(m) {
			return false
		}
	}
	return matchFound
}
