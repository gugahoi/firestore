package api

import "testing"

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"relative", "users/u1", "projects/proj/databases/(default)/documents/users/u1"},
		{"relative with leading slash", "/users/u1", "projects/proj/databases/(default)/documents/users/u1"},
		{"relative with query", "users?documentId=u1", "projects/proj/databases/(default)/documents/users?documentId=u1"},
		{"verbatim passthrough", "projects/other/databases/(default)/documents/users/u1", "projects/other/databases/(default)/documents/users/u1"},
		{"project placeholder", "projects/{project}/databases/{database}/documents:runQuery", "projects/proj/databases/(default)/documents:runQuery"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvePath("proj", "(default)", tt.path); got != tt.want {
				t.Errorf("resolvePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultMethod(t *testing.T) {
	tests := []struct {
		method  string
		hasBody bool
		want    string
	}{
		{"", false, "GET"},
		{"", true, "POST"},
		{"delete", false, "DELETE"},
		{"PATCH", true, "PATCH"},
	}
	for _, tt := range tests {
		if got := defaultMethod(tt.method, tt.hasBody); got != tt.want {
			t.Errorf("defaultMethod(%q, %v) = %q, want %q", tt.method, tt.hasBody, got, tt.want)
		}
	}
}
