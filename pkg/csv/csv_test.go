package csv

import (
	"bytes"
	"io/ioutil"
	"testing"
)

type FakeReadFiler struct {
	Str string
}

// ReadFile here's a fake ReadFile method that matches the signature of ioutil.ReadFile
func (f FakeReadFiler) ReadFile(filename string) ([]byte, error) {
	buf := bytes.NewBufferString(f.Str)
	return ioutil.ReadAll(buf)
}

func Test_readDateValue(t *testing.T) {
	type args struct {
		date string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test '01/02/2023' date values",
			args: args{date: "01/02/2023"},
			want: "01/02/23",
		},
		{
			name: "Test '2023-01-02' date values",
			args: args{date: "2023-01-02"},
			want: "01/02/23",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readDateValue(tt.args.date); got != tt.want {
				t.Errorf("readDateValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func checkTestingError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("error: %s\n", err.Error())
	}
}
