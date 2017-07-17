package common

import (
	"testing"
	"time"
)

func TestFilter(t *testing.T) {
	lm := LogMessage{
		Timestamp: time.Now(),
		Message:   "Sup",
		Attributes: map[string]interface{}{
			"someStr": "Zef",
			"someN":   34,
		},
	}
	lastHour := time.Now().Add(-time.Hour)
	nextHour := time.Now().Add(time.Hour)
	shouldMatchQuery := Query{
		Filters: []QueryFilter{
			QueryFilter{FieldName: "someStr", Value: "Zef"},
		},
	}
	shouldMatchQuery2 := Query{
		Filters: []QueryFilter{
			QueryFilter{FieldName: "someN", Value: "34"},
		},
	}
	shouldMatchQuery3 := Query{
		QueryString: "zef",
		Filters: []QueryFilter{
			QueryFilter{FieldName: "someN", Value: "34"},
		},
	}
	shouldMatchQuery4 := Query{
		QueryString: "zef",
		Filters: []QueryFilter{
			QueryFilter{FieldName: "someN", Value: "34"},
		},
		Before: &nextHour,
		After:  &lastHour,
	}
	shouldNotMatchQuery := Query{
		Filters: []QueryFilter{
			QueryFilter{FieldName: "someStr", Value: "Pete"},
		},
	}
	shouldNotMatchQuery2 := Query{
		QueryString: "bla",
		Filters: []QueryFilter{
			QueryFilter{FieldName: "someStr", Value: "Pete"},
		},
	}
	shouldNotMatchQuery3 := Query{
		After: &nextHour,
	}
	if !MatchesQuery(lm, shouldMatchQuery) {
		t.Errorf("Did not match")
	}
	if !MatchesQuery(lm, shouldMatchQuery2) {
		t.Errorf("Did not match 2")
	}
	if !MatchesQuery(lm, shouldMatchQuery3) {
		t.Errorf("Did not match 3")
	}
	if !MatchesQuery(lm, shouldMatchQuery4) {
		t.Errorf("Did not match 4")
	}
	if MatchesQuery(lm, shouldNotMatchQuery) {
		t.Errorf("Did match")
	}
	if MatchesQuery(lm, shouldNotMatchQuery2) {
		t.Errorf("Did match 2")
	}
	if MatchesQuery(lm, shouldNotMatchQuery3) {
		t.Errorf("Did match 3")
	}
}

func TestFlatten(t *testing.T) {
	into := make(map[string]interface{})
	inputJsonString := `{
		"name": "Zef Hemel",
		"age": 34,
		"docker": {
			"service": "myservice",
			"container": "something else",
			"deeper": {
				"b": 10
			}
		}
	}`
	expectedJsonString := `{
		"name": "Zef Hemel",
		"age": 34,
		"docker.service": "myservice",
		"docker.container": "something else",
		"docker.deeper.b": 10
	}`
	var inputObj map[string]interface{}
	var expectedObj map[string]interface{}
	MustJsonDecode(inputJsonString, &inputObj)
	MustJsonDecode(expectedJsonString, &expectedObj)
	FlattenAttributes(inputObj, into, "")

	if MustJsonEncode(into) != MustJsonEncode(expectedObj) {
		t.Error("Didn't match")
	}
}
