package browse

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type QueryParam struct {
	Field    string
	Operator string
	Value    interface{}
}

func (q QueryParam) String() string {
	switch v := q.Value.(type) {
	case string:
		return fmt.Sprintf("%s %s %q", q.Field, q.Operator, v)
	default:
		return fmt.Sprintf("%s %s %v", q.Field, q.Operator, q.Value)
	}
}

var validOperators = map[string]bool{
	"==": true, "!=": true,
	"<": true, ">": true,
	"<=": true, ">=": true,
	"array-contains": true, "in": true,
}

// ParseQueryArgs parses :query arguments into a QueryParam.
func ParseQueryArgs(args []string) (QueryParam, error) {
	if len(args) < 3 {
		return QueryParam{}, fmt.Errorf("usage: :query <field> <op> <value>")
	}

	field := args[0]
	op := args[1]

	if !validOperators[op] {
		return QueryParam{}, fmt.Errorf("invalid operator %q, use: ==, !=, <, >, <=, >=, array-contains, in", op)
	}

	valueStr := strings.Join(args[2:], " ")
	value, err := ParseValue(valueStr)
	if err != nil {
		return QueryParam{}, fmt.Errorf("invalid value: %w", err)
	}

	return QueryParam{Field: field, Operator: op, Value: value}, nil
}

// ParseValue detects the type of a value string and returns the Go representation.
func ParseValue(s string) (interface{}, error) {
	s = strings.TrimSpace(s)

	if s == "true" {
		return true, nil
	}
	if s == "false" {
		return false, nil
	}

	// Quoted string
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var v string
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return nil, err
		}
		return v, nil
	}

	// JSON array
	if len(s) >= 2 && s[0] == '[' {
		var arr []interface{}
		if err := json.Unmarshal([]byte(s), &arr); err != nil {
			return nil, fmt.Errorf("invalid array: %w", err)
		}
		return arr, nil
	}

	// Number
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return n, nil
	}

	return nil, fmt.Errorf("unrecognized value %q — use quotes for strings", s)
}
