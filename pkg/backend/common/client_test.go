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
		{
			EqualityFilters: []EqualityFilter{
				{FieldName: "someStr", Value: "Zef", Operator: "="},
			},
		},
		{
			EqualityFilters: []EqualityFilter{
				{FieldName: "someN", Value: "34", Operator: "="},
			},
		},
		{
			QueryString: "zef",
			EqualityFilters: []EqualityFilter{
				{FieldName: "someN", Value: "34", Operator: "="},
			},
		},
		{
			QueryString: "zef",
			EqualityFilters: []EqualityFilter{
				{FieldName: "someN", Value: "34", Operator: "="},
			},
			Before: &nextHour,
			After:  &lastHour,
		},
		{
			EqualityFilters: []EqualityFilter{
				{FieldName: "someN", Value: "32", Operator: "!="},
			},
		},
		{
			EqualityFilters: []EqualityFilter{
				{FieldName: "someNonexistingField", Value: "Pete", Operator: "!="},
			},
		},
		{
			ExistenceFilters: []ExistenceFilter{
				{
					FieldName: "message",
					Exists:    true,
				},
			},
		},
		{
			ExistenceFilters: []ExistenceFilter{
				{
					FieldName: "message",
					Exists:    true,
				},
				{
					FieldName: "someStr",
					Exists:    true,
				},
			},
		},
		{
			ExistenceFilters: []ExistenceFilter{
				{
					FieldName: "message",
					Exists:    true,
				},
				{
					FieldName: "someStr",
					Exists:    true,
				},
				{
					FieldName: "bar",
					Exists:    false,
				},
			},
		},
		{
			MembershipFilters: []MembershipFilter{
				{
					FieldName:   "message",
					ValidValues: []string{"Sup", "bar"},
				},
			},
		},
		{
			MembershipFilters: []MembershipFilter{
				{
					FieldName:     "message",
					InvalidValues: []string{"foo", "bar"},
				},
			},
		},
		{
			MembershipFilters: []MembershipFilter{
				{
					FieldName:   "someN",
					ValidValues: []string{"Sup", "34"},
				},
			},
		},
	}
	shouldNotMatchQueries := []Query{
		{
			EqualityFilters: []EqualityFilter{
				{FieldName: "someStr", Value: "Pete", Operator: "="},
			},
		},
		{
			QueryString: "bla",
			EqualityFilters: []EqualityFilter{
				{FieldName: "someStr", Value: "Pete", Operator: "="},
			},
		},
		{
			After: &nextHour,
		},
		{
			ExistenceFilters: []ExistenceFilter{
				{
					FieldName: "message",
					Exists:    true,
				},
				{
					FieldName: "someStr",
					Exists:    false,
				},
				{
					FieldName: "bar",
					Exists:    false,
				},
			},
		},
		{
			ExistenceFilters: []ExistenceFilter{
				{
					FieldName: "bar",
					Exists:    true,
				},
			},
		},
		{
			MembershipFilters: []MembershipFilter{
				{
					FieldName:   "someN",
					ValidValues: []string{"45", "bar"},
				},
			},
		},
		{
			MembershipFilters: []MembershipFilter{
				{
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

func TestMembershipFilter_Matches(t *testing.T) {
	type fields struct {
		FieldName     string
		ValidValues   []string
		InvalidValues []string
	}
	tests := []struct {
		name   string
		fields fields
		m      LogMessage
		want   bool
	}{
		{
			name: "Enforce field existence for inclusive member filters",
			fields: fields{
				FieldName: "domain",
				ValidValues: []string{
					"ax",
				},
				InvalidValues: []string{},
			},
			m: LogMessage{
				Attributes: map[string]interface{}{
					"not-domain": "foo",
				},
			},
			want: false,
		},
		{
			name: "Don't enforce field existence for exclusive-only filter",
			fields: fields{
				FieldName:   "domain",
				ValidValues: []string{},
				InvalidValues: []string{
					"ax",
				},
			},
			m: LogMessage{
				Attributes: map[string]interface{}{
					"not-domain": "foo",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := MembershipFilter{
				FieldName:     tt.fields.FieldName,
				ValidValues:   tt.fields.ValidValues,
				InvalidValues: tt.fields.InvalidValues,
			}
			if got := f.Matches(tt.m); got != tt.want {
				t.Errorf("MembershipFilter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
