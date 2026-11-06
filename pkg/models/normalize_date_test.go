package models

import (
	"testing"
)

func Test_readDateValue(t *testing.T) {
	type args struct {
		dateStr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test 01/02/03",
			args: args{dateStr: "01/02/03"},
			want: "01/02/03",
		},
		{
			name: "Test '01/02/2023' date values",
			args: args{dateStr: "01/02/2023"},
			want: "01/02/23",
		},
		{
			name: "Test 1/2/03",
			args: args{dateStr: "1/2/03"},
			want: "01/02/03",
		},
		{
			name: "Test 1/2/2003",
			args: args{dateStr: "1/2/2003"},
			want: "01/02/03",
		},
		{
			name: "Test 2023-01-02",
			args: args{dateStr: "2023-01-02"},
			want: "01/02/23",
		},
		{
			name: "Test 2023-1-2",
			args: args{dateStr: "2023-1-2"},
			want: "01/02/23",
		},
		{
			name: "Test empty string",
			args: args{dateStr: ""},
			want: "",
		},
		{
			name: "Test 'not a date'",
			args: args{dateStr: "not a date"},
			want: "",
		},
		{
			name: "Test '2023/01/02' slashes year-first",
			args: args{dateStr: "2023/01/02"},
			want: "",
		},
		{
			name: "Test '01-02-2023' dashes year-last",
			args: args{dateStr: "01-02-2023"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadDateValue(tt.args.dateStr); got != tt.want {
				t.Errorf("readDateValue(%s) = %+v, want %+v", tt.name, got, tt.want)
			}
		})
	}
}
