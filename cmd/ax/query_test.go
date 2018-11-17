package main

import (
	"reflect"
	"sort"
	"testing"
	"time"

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

func setupTestTime(testTime time.Time) func() {
	timeNow = func() time.Time {
		return testTime
	}
	return func() { timeNow = time.Now }
}

func Test_humanToTimeInterval(t *testing.T) {
	testTime := time.Date(2018, 10, 1, 12, 15, 30, 0, time.UTC)

	testCases := []struct {
		input     string
		wantAfter time.Time
	}{
		{"1 day", time.Date(2018, 9, 30, 12, 15, 30, 0, time.UTC)},
		{"0 days", time.Date(2018, 10, 1, 12, 15, 30, 0, time.UTC)},
		{"31 days", time.Date(2018, 8, 31, 12, 15, 30, 0, time.UTC)},
		{"31 d", time.Date(2018, 8, 31, 12, 15, 30, 0, time.UTC)},
		{"1 hour", time.Date(2018, 10, 1, 11, 15, 30, 0, time.UTC)},
		{"1 h", time.Date(2018, 10, 1, 11, 15, 30, 0, time.UTC)},
		{"4 hours", time.Date(2018, 10, 1, 8, 15, 30, 0, time.UTC)},
		{"2 years", time.Date(2016, 10, 1, 12, 15, 30, 0, time.UTC)},
		{"1 year", time.Date(2017, 10, 1, 12, 15, 30, 0, time.UTC)},
		{"1 y", time.Date(2017, 10, 1, 12, 15, 30, 0, time.UTC)},
		{"1 months", time.Date(2018, 9, 1, 12, 15, 30, 0, time.UTC)},
		{"1 month", time.Date(2018, 9, 1, 12, 15, 30, 0, time.UTC)},
		{"1 m", time.Date(2018, 9, 1, 12, 15, 30, 0, time.UTC)},
		{"30 minutes", time.Date(2018, 10, 1, 11, 45, 30, 0, time.UTC)},
		{"1 minutes", time.Date(2018, 10, 1, 12, 14, 30, 0, time.UTC)},
		{"1 minute", time.Date(2018, 10, 1, 12, 14, 30, 0, time.UTC)},
		{"1 min", time.Date(2018, 10, 1, 12, 14, 30, 0, time.UTC)},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(*testing.T) {
			teardown := setupTestTime(testTime)
			defer teardown()
			before, after, _ := lastToTimeInterval(tc.input)

			if *before != testTime {
				// it should stay constant through all cases
				t.Fatalf("exepcted before to equal now, got %v", *before)
			}

			if *after != tc.wantAfter {
				t.Fatalf("exepcted after to be %v, got %v", tc.wantAfter, *after)
			}
		})
	}
}
