package main

import (
	"reflect"
	"testing"

	"github.com/egnyte/ax/pkg/backend/common"
)

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
			if got := buildMembershipFilters(tt.args.oneOfs, tt.args.notOneOfs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildMembershipFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}
