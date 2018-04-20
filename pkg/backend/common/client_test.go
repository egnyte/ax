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
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someStr", Value: "Zef", Operator: "="},
			},
		},
		Query{
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someN", Value: "34", Operator: "="},
			},
		},
		Query{
			QueryString: "zef",
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someN", Value: "34", Operator: "="},
			},
		},
		Query{
			QueryString: "zef",
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someN", Value: "34", Operator: "="},
			},
			Before: &nextHour,
			After:  &lastHour,
		},
		Query{
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someN", Value: "32", Operator: "!="},
			},
		},
		Query{
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someNonexistingField", Value: "Pete", Operator: "!="},
			},
		},
		Query{
			ExistenceFilters: []ExistenceFilter{
				ExistenceFilter{
					FieldName: "message",
					Exists:    true,
				},
			},
		},
		Query{
			ExistenceFilters: []ExistenceFilter{
				ExistenceFilter{
					FieldName: "message",
					Exists:    true,
				},
				ExistenceFilter{
					FieldName: "someStr",
					Exists:    true,
				},
			},
		},
		Query{
			ExistenceFilters: []ExistenceFilter{
				ExistenceFilter{
					FieldName: "message",
					Exists:    true,
				},
				ExistenceFilter{
					FieldName: "someStr",
					Exists:    true,
				},
				ExistenceFilter{
					FieldName: "bar",
					Exists:    false,
				},
			},
		},
		Query{
			MembershipFilters: []MembershipFilter{
				MembershipFilter{
					FieldName:   "message",
					ValidValues: []string{"Sup", "bar"},
				},
			},
		},
		Query{
			MembershipFilters: []MembershipFilter{
				MembershipFilter{
					FieldName:     "message",
					InvalidValues: []string{"foo", "bar"},
				},
			},
		},
		Query{
			MembershipFilters: []MembershipFilter{
				MembershipFilter{
					FieldName:   "someN",
					ValidValues: []string{"Sup", "34"},
				},
			},
		},
	}
	shouldNotMatchQueries := []Query{
		Query{
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someStr", Value: "Pete", Operator: "="},
			},
		},
		Query{
			QueryString: "bla",
			EqualityFilters: []EqualityFilter{
				EqualityFilter{FieldName: "someStr", Value: "Pete", Operator: "="},
			},
		},
		Query{
			After: &nextHour,
		},
		Query{
			ExistenceFilters: []ExistenceFilter{
				ExistenceFilter{
					FieldName: "message",
					Exists:    true,
				},
				ExistenceFilter{
					FieldName: "someStr",
					Exists:    false,
				},
				ExistenceFilter{
					FieldName: "bar",
					Exists:    false,
				},
			},
		},
		Query{
			ExistenceFilters: []ExistenceFilter{
				ExistenceFilter{
					FieldName: "bar",
					Exists:    true,
				},
			},
		},
		Query{
			MembershipFilters: []MembershipFilter{
				MembershipFilter{
					FieldName:   "someN",
					ValidValues: []string{"45", "bar"},
				},
			},
		},
		Query{
			MembershipFilters: []MembershipFilter{
				MembershipFilter{
					FieldName:     "someN",
					InvalidValues: []string{"34"},
				},
			},
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
