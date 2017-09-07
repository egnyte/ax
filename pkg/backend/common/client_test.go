package common

import (
	"testing"
	"time"
)

func TestFilter(t *testing.T) {
	lm := LogMessage{
		Timestamp: time.Now(),
		Attributes: map[string]interface{}{
			"message": "Sup",
			"someStr": "Zef",
			"someN":   34,
		},
	}
	lastHour := time.Now().Add(-time.Hour)
	nextHour := time.Now().Add(time.Hour)
	shouldMatchQueries := []Query{
		Query{
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someStr", Value: "Zef", Operator: "="},
			},
		},
		Query{
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someN", Value: "34", Operator: "="},
			},
		},
		Query{
			QueryString: "zef",
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someN", Value: "34", Operator: "="},
			},
		},
		Query{
			QueryString: "zef",
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someN", Value: "34", Operator: "="},
			},
			Before: &nextHour,
			After:  &lastHour,
		},
		Query{
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someN", Value: "32", Operator: "!="},
			},
		},
		Query{
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someNonexistingField", Value: "Pete", Operator: "!="},
			},
		},
	}
	shouldNotMatchQueries := []Query{
		Query{
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someStr", Value: "Pete", Operator: "="},
			},
		},
		Query{
			QueryString: "bla",
			Filters: []QueryFilter{
				QueryFilter{FieldName: "someStr", Value: "Pete", Operator: "="},
			},
		},
		Query{
			After: &nextHour,
		},
	}
	for i, shouldMatch := range shouldMatchQueries {
		if !MatchesQuery(lm, shouldMatch) {
			t.Errorf("Did not match: %d: %+v", i, shouldMatch)
		}
	}
	for i, shouldNotMatch := range shouldNotMatchQueries {
		if MatchesQuery(lm, shouldNotMatch) {
			t.Errorf("Did match: %d: %+v", i, shouldNotMatch)
		}
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
