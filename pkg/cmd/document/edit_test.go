package document

import (
	"reflect"
	"testing"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		in   string
		want any
	}{
		{"foo", "foo"},
		{"42", float64(42)},
		{"3.14", 3.14},
		{"true", true},
		{"false", false},
		{`["a","b"]`, []any{"a", "b"}},
		{`"42"`, "42"}, // quotes force a string
		{"NYC", "NYC"},
	}
	for _, tt := range tests {
		got := parseValue(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseValue(%q) = %#v, want %#v", tt.in, got, tt.want)
		}
	}
}

func TestParseSetArg(t *testing.T) {
	u, err := parseSetArg("address.city=NYC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Path != "address.city" || u.Value != "NYC" {
		t.Errorf("parseSetArg = {Path:%q Value:%v}, want {address.city NYC}", u.Path, u.Value)
	}

	for _, bad := range []string{"noequals", "=value"} {
		if _, err := parseSetArg(bad); err == nil {
			t.Errorf("parseSetArg(%q) expected error, got nil", bad)
		}
	}
}
