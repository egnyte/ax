package common

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var TimeFormat = time.RFC3339

type Client interface {
	Query(ctx context.Context, query Query) <-chan LogMessage
}

type QueryFilter struct {
	FieldName string
	Operator  string
	Value     string
}

type Query struct {
	QueryString  string
	After        *time.Time
	Before       *time.Time
	SelectFields []string
	Filters      []QueryFilter
	MaxResults   int
	Unique       bool
	Follow       bool
}

type QuerySelectors struct {
	Before      string   `yaml:"before,omitempty"`
	After       string   `yaml:"after,omitempty"`
	Select      []string `yaml:"select,omitempty"`
	Where       []string `yaml:"where,omitempty"`
	Unique      bool     `yaml:"unique,omitempty"`
	QueryString []string `yaml:"query,omitempty"`
}

type LogMessage struct {
	ID         string                 `json:"id,omitempty"`
	Timestamp  time.Time              `json:"@timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

// Performs a shallow copy of the Attributes map and adds fields for '@id' and '@timestamp'
func (lm LogMessage) Map() map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range lm.Attributes {
		out[k] = v
	}
	if lm.ID != "" {
		out["@id"] = lm.ID
	}
	out["@timestamp"] = lm.Timestamp.Format(TimeFormat)
	return out
}

func (lm LogMessage) UniqueID() string {
	if lm.ID != "" {
		return lm.ID
	}
	// Didn't get a unique ID from our source, let's just SHA1 the message itself
	m := lm.Map()
	h := sha1.New()
	encoder := json.NewEncoder(h)
	encoder.Encode(&m)
	return fmt.Sprintf("%x", h.Sum(nil))[0:10]
}

func (lm LogMessage) ContentHash() string {
	h := sha1.New()
	encoder := json.NewEncoder(h)
	encoder.Encode(&lm.Attributes)
	return fmt.Sprintf("%x", h.Sum(nil))[0:10]
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
	newMessage.ID = message.ID
	newMessage.Timestamp = message.Timestamp
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
	switch f.Operator {
	case "=":
		return ok && f.Value == fmt.Sprintf("%v", val)
	case "!=":
		return !ok || (ok && f.Value != fmt.Sprintf("%v", val))
	default:
		panic("Not supported operator")
	}
}

func matchesPhrase(s, phrase string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(phrase))
}

func MatchesQuery(m LogMessage, q Query) bool {
	msg, _ := m.Attributes["message"].(string)
	matchFound := matchesPhrase(msg, q.QueryString)
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
