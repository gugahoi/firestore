package browse

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportJSON(t *testing.T) {
	docs := []map[string]interface{}{
		{"name": "Alice", "age": float64(30)},
		{"name": "Bob", "age": float64(25)},
	}

	result, err := ExportJSON(docs)
	if err != nil {
		t.Fatal(err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("expected 2 docs, got %d", len(parsed))
	}
	if parsed[0]["name"] != "Alice" {
		t.Errorf("expected Alice, got %v", parsed[0]["name"])
	}
}

func TestExportJSONEmpty(t *testing.T) {
	result, err := ExportJSON([]map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "[]" {
		t.Errorf("expected [], got %q", result)
	}
}

func TestExportJSONSingleDoc(t *testing.T) {
	docs := []map[string]interface{}{
		{"key": "value"},
	}
	result, err := ExportJSON(docs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, `"key"`) {
		t.Errorf("expected key in result: %s", result)
	}
}

func TestExportNDJSON(t *testing.T) {
	docs := []map[string]interface{}{
		{"name": "Alice"},
		{"name": "Bob"},
	}

	result, err := ExportNDJSON(docs)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for _, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line is not valid JSON: %q, err: %v", line, err)
		}
	}
}

func TestExportNDJSONEmpty(t *testing.T) {
	result, err := ExportNDJSON([]map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestExportJSONNestedData(t *testing.T) {
	docs := []map[string]interface{}{
		{
			"user": map[string]interface{}{
				"name":  "Alice",
				"roles": []interface{}{"admin", "user"},
			},
			"active": true,
			"count":  float64(42),
		},
	}

	result, err := ExportJSON(docs)
	if err != nil {
		t.Fatal(err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	user := parsed[0]["user"].(map[string]interface{})
	if user["name"] != "Alice" {
		t.Errorf("expected nested name Alice, got %v", user["name"])
	}
}
