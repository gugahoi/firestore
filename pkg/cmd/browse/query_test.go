package browse

import (
	"testing"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		input string
		want  interface{}
	}{
		{`"active"`, "active"},
		{`"hello world"`, "hello world"},
		{`42`, float64(42)},
		{`3.14`, float64(3.14)},
		{`true`, true},
		{`false`, false},
		{`["a","b","c"]`, []interface{}{"a", "b", "c"}},
		{`[1,2,3]`, []interface{}{float64(1), float64(2), float64(3)}},
	}

	for _, tt := range tests {
		got, err := ParseValue(tt.input)
		if err != nil {
			t.Errorf("ParseValue(%q) error: %v", tt.input, err)
			continue
		}

		switch expected := tt.want.(type) {
		case string:
			if got != expected {
				t.Errorf("ParseValue(%q) = %v, want %v", tt.input, got, expected)
			}
		case float64:
			if got != expected {
				t.Errorf("ParseValue(%q) = %v, want %v", tt.input, got, expected)
			}
		case bool:
			if got != expected {
				t.Errorf("ParseValue(%q) = %v, want %v", tt.input, got, expected)
			}
		case []interface{}:
			gotArr, ok := got.([]interface{})
			if !ok {
				t.Errorf("ParseValue(%q) not an array", tt.input)
				continue
			}
			if len(gotArr) != len(expected) {
				t.Errorf("ParseValue(%q) array len = %d, want %d", tt.input, len(gotArr), len(expected))
			}
		}
	}
}

func TestParseValueErrors(t *testing.T) {
	tests := []string{
		"unquoted",
		"[invalid",
	}

	for _, input := range tests {
		_, err := ParseValue(input)
		if err == nil {
			t.Errorf("ParseValue(%q) expected error", input)
		}
	}
}

func TestParseQueryArgs(t *testing.T) {
	tests := []struct {
		args     []string
		wantOp   string
		wantErr  bool
	}{
		{[]string{"status", "==", `"active"`}, "==", false},
		{[]string{"count", ">", "10"}, ">", false},
		{[]string{"active", "==", "true"}, "==", false},
		{[]string{"tags", "array-contains", `"urgent"`}, "array-contains", false},
		{[]string{"status", "in", `["active","pending"]`}, "in", false},
		{[]string{"field"}, "", true},
		{[]string{"field", "INVALID", "val"}, "", true},
	}

	for _, tt := range tests {
		q, err := ParseQueryArgs(tt.args)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseQueryArgs(%v) expected error", tt.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseQueryArgs(%v) error: %v", tt.args, err)
			continue
		}
		if q.Operator != tt.wantOp {
			t.Errorf("ParseQueryArgs(%v) op = %q, want %q", tt.args, q.Operator, tt.wantOp)
		}
	}
}

func TestQueryParamString(t *testing.T) {
	q := QueryParam{Field: "status", Operator: "==", Value: "active"}
	got := q.String()
	want := `status == "active"`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	q2 := QueryParam{Field: "count", Operator: ">", Value: float64(10)}
	got2 := q2.String()
	want2 := "count > 10"
	if got2 != want2 {
		t.Errorf("String() = %q, want %q", got2, want2)
	}
}
