package parser

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name   string
		lines  [][]byte
		expect [][]byte
	}{
		{
			name:   "PING command",
			lines:  [][]byte{[]byte("*1"), []byte("PING")},
			expect: [][]byte{[]byte("PING")},
		},
		{
			name:   "ECHO command",
			lines:  [][]byte{[]byte("*2"), []byte("ECHO"), []byte("$3"), []byte("hey")},
			expect: [][]byte{[]byte("ECHO"), []byte("hey")},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse(c.lines)
			if !reflect.DeepEqual(got, c.expect) {
				t.Errorf("Parse() = %v, want %v", got, c.expect)
			}
		})
	}
}
