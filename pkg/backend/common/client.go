package common

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	TimeFormat        = time.RFC3339
	FollowPollTime    = 5 * time.Second
	ConnectionRetries = 10
)

type Client interface {
	Query(ctx context.Context, query Query) <-chan LogMessage
	ImplementsAdvancedFilters() bool
}

type EqualityFilter struct {
	FieldName string
	Operator  string
	Value     string
}

type ExistenceFilter struct {
	FieldName string
	Exists    bool // true if FieldName should exist in the message, false if FieldName should *not* exist

}
type MembershipFilter struct {
	FieldName     string
	ValidValues   []string
	InvalidValues []string
}

type Query struct {
	QueryString       string
	After             *time.Time
	Before            *time.Time
	SelectFields      []string
	EqualityFilters   []EqualityFilter
	ExistenceFilters  []ExistenceFilter
	MembershipFilters []MembershipFilter
	MaxResults        int
	Unique            bool
	Follow            bool
}

type QuerySelectors struct {
	Before      string   `yaml:"before,omitempty"`
	After       string   `yaml:"after,omitempty"`
	Select      []string `yaml:"select,omitempty"`
	Where       []string `yaml:"where,omitempty"`
	OneOf       []string `yaml:"one_of,omitempty"`
	NotOneOf    []string `yaml:"not_one_of,omitempty"`
	Exists      []string `yaml:"exists,omitempty"`
	NotExists   []string `yaml:"not_exists,omitempty"`
	Unique      bool     `yaml:"unique,omitempty"`
	QueryString []string `yaml:"query,omitempty"`
}

type LogMessage struct {
	ID        string    `json:"id,omitempty"`
	Timestamp time.Time `json:"@timestamp"`
	// required: "message" attribute
	Attributes map[string]interface{} `json:"attributes"`
}

// Map performs a shallow copy of the Attributes map and adds fields for '@id' and '@timestamp'
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
		log.Printf("Could not JSON encode: %+v\n", obj)
		return "{}"
	}
	return string(buf)
}

func MustJsonDecode(jsonString string, dst interface{}) {
	decoder := json.NewDecoder(strings.NewReader(jsonString))
	err := decoder.Decode(dst)
	if err != nil {
		log.Fatal(err)
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

func (f EqualityFilter) Matches(m LogMessage) bool {
	val, ok := m.Attributes[f.FieldName]
	switch f.Operator {
	case "=":
		return ok && f.Value == fmt.Sprintf("%v", val)
	case "!=":
		return !ok || (ok && f.Value != fmt.Sprintf("%v", val))
	default:
		fmt.Printf("Not supported operatior: %s\n", f.Operator)
		return false
	}
}

// Matches indicates whether the existence filter matches the log message
func (f ExistenceFilter) Matches(m LogMessage) bool {
	_, ok := m.Attributes[f.FieldName]
	// true if we expect the field and it exists, or if we don't expect it and it doesn't exist
	return (f.Exists && ok) || (!f.Exists && !ok)
}

func isStringInSlice(needle string, haystack []string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// Matches indicates whether the log message satisfies membership constraints
func (f MembershipFilter) Matches(m LogMessage) bool {
	valueInterface, ok := m.Attributes[f.FieldName]
	// If value is missing, only match if no members are expected
	if !ok {
		return len(f.ValidValues) == 0
	}
	valueAsString := fmt.Sprintf("%v", valueInterface)
	// Only perform membership checks for the respective kind of contraint if any constraint is specified
	return (len(f.ValidValues) == 0 || isStringInSlice(valueAsString, f.ValidValues)) && (len(f.InvalidValues) == 0 || !isStringInSlice(valueAsString, f.InvalidValues))
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
	for _, f := range q.EqualityFilters {
		if !f.Matches(m) {
			return false
		}
	}
	for _, f := range q.MembershipFilters {
		if !f.Matches(m) {
			return false
		}
	}
	for _, f := range q.ExistenceFilters {
		if !f.Matches(m) {
			return false
		}
	}
	return matchFound
}

// This is a simplistic way to implement log "following" (tailing) in a generic way.
// The idea is to simply execute the fetch log query (implemented by `queryMessagesFunc`) over and over, every `FollowPollTime` seconds,
// and push the results through the deduplicating result channel returned (deduplication happens based on message ID)
// This may seem overly inefficient, but due to the eventual-consistency type behavior of many log aggregation systems, logs may not actually
// arrive in sequence, so requesting new logs based on timestamps won't work reliably.
func ReQueryFollow(ctx context.Context, queryMessagesFunc func() ([]LogMessage, error)) <-chan LogMessage {
	resultChan := make(chan LogMessage)
	go func() {
		retries := 0
		for {
			select {
			case <-ctx.Done():
				close(resultChan)
				return
			default:
			}
			allMessages, err := queryMessagesFunc()
			select {
			case <-ctx.Done():
				close(resultChan)
				return
			default:
			}
			if err != nil {
				fmt.Println(err)
				retries++
				select {
				case <-ctx.Done():
					close(resultChan)
					return
				default:
				}
				if retries < ConnectionRetries {
					fmt.Fprintf(os.Stderr, "Could not connect: %v retrying in 5s\n", err)
					if canceableSleep(ctx, FollowPollTime) {
						// Canceled
						close(resultChan)
						return
					}
					continue
				} else {
					fmt.Fprintf(os.Stderr, "Could not connect: %v\nExceeded total number of retries, exiting.\n", err)
					close(resultChan)
					return
				}
			}
			// Request succesful, so reset retry count
			retries = 0
			for _, message := range allMessages {
				resultChan <- message
			}
			if canceableSleep(ctx, FollowPollTime) {
				close(resultChan)
				return
			}
		}
	}()
	return Dedup(resultChan)
}

// Returns if canceled
func canceableSleep(ctx context.Context, duration time.Duration) bool {
	select {
	case <-time.After(duration):
		return false
	case <-ctx.Done():
		return true
	}
}
