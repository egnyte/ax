package main

import (
	"reflect"
	"sort"
	"testing"

	"github.com/egnyte/ax/pkg/backend/common"
)

func sortMembershipFilters(slice []common.MembershipFilter) []common.MembershipFilter {
	sort.SliceStable(slice, func(i, j int) bool {
		a, b := slice[i], slice[j]

		// Sort internal strings so that equivalent filters are equal
		sort.Strings(a.InvalidValues)
		sort.Strings(b.InvalidValues)
		sort.Strings(a.ValidValues)
		sort.Strings(b.InvalidValues)

		return sort.StringsAreSorted([]string{a.FieldName, b.FieldName})
	})
	return slice
}

func Test_buildMembershipFilters(t *testing.T) {
	type args struct {
		oneOfs    []string
		notOneOfs []string
	}
	tests := []struct {
		name string
		args args
		want []common.MembershipFilter
	}{
		{
			name: "Simple inclusive test",
			args: args{
				oneOfs:    []string{"foo:bar"},
				notOneOfs: []string{},
			},
			want: []common.MembershipFilter{
				{
					FieldName:     "foo",
					ValidValues:   []string{"bar"},
					InvalidValues: []string{},
				},
			},
		},
		{
			name: "Complex filter",
			args: args{
				oneOfs:    []string{"foo:bar", "foo:burp"},
				notOneOfs: []string{"fizz:bang"},
			},
			want: []common.MembershipFilter{
				{
					FieldName:     "foo",
					ValidValues:   []string{"bar", "burp"},
					InvalidValues: []string{},
				},
				{
					FieldName:     "fizz",
					ValidValues:   []string{},
					InvalidValues: []string{"bang"},
				},
			},
		},
		{
			name: "Simple exclusive test",
			args: args{
				oneOfs:    []string{},
				notOneOfs: []string{"fizz:bang"},
			},
			want: []common.MembershipFilter{
				{
					FieldName:     "fizz",
					ValidValues:   []string{},
					InvalidValues: []string{"bang"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildMembershipFilters(tt.args.oneOfs, tt.args.notOneOfs); !reflect.DeepEqual(sortMembershipFilters(got), sortMembershipFilters(tt.want)) {
				t.Errorf("buildMembershipFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildExistenceFilters(t *testing.T) {
	type args struct {
		exists    []string
		notExists []string
	}
	tests := []struct {
		name string
		args args
		want []common.ExistenceFilter
	}{
		{
			name: "Simple test",
			args: args{
				exists:    []string{"foo"},
				notExists: []string{"bar"},
			},
			want: []common.ExistenceFilter{
				{
					FieldName: "foo",
					Exists:    true,
				},
				{
					FieldName: "bar",
					Exists:    false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildExistenceFilters(tt.args.exists, tt.args.notExists); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildExistenceFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}
