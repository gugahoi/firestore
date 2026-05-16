package browse

import (
	"testing"
)

func TestFilterItems(t *testing.T) {
	items := []ListItem{
		{id: "users"},
		{id: "orders"},
		{id: "user-profiles"},
		{id: "Products"},
		{id: "ADMIN_USERS"},
	}

	tests := []struct {
		pattern string
		want    []string
	}{
		{"", []string{"users", "orders", "user-profiles", "Products", "ADMIN_USERS"}},
		{"user", []string{"users", "user-profiles", "ADMIN_USERS"}},
		{"USER", []string{"users", "user-profiles", "ADMIN_USERS"}},
		{"prod", []string{"Products"}},
		{"order", []string{"orders"}},
		{"xyz", nil},
		{"admin", []string{"ADMIN_USERS"}},
	}

	for _, tt := range tests {
		got := FilterItems(items, tt.pattern)
		gotIDs := make([]string, len(got))
		for i, item := range got {
			gotIDs[i] = item.id
		}
		if len(got) != len(tt.want) {
			t.Errorf("FilterItems(%q) = %v, want %v", tt.pattern, gotIDs, tt.want)
			continue
		}
		for i := range got {
			if got[i].id != tt.want[i] {
				t.Errorf("FilterItems(%q)[%d] = %q, want %q", tt.pattern, i, got[i].id, tt.want[i])
			}
		}
	}
}

func TestFilterSections(t *testing.T) {
	sections := []Section{
		{
			title: "Documents",
			items: []ListItem{
				{id: "doc1"},
				{id: "doc2"},
				{id: "other"},
			},
		},
		{
			title: "Subcollections",
			items: []ListItem{
				{id: "sub-doc"},
				{id: "archive"},
			},
		},
		{
			title: "Data",
			items: []ListItem{
				{id: "field1"},
			},
		},
	}

	filtered := FilterSections(sections, "doc")
	if len(filtered) != 3 {
		t.Fatalf("FilterSections returned %d sections, want 3", len(filtered))
	}

	// Documents section should be filtered
	if len(filtered[0].items) != 2 {
		t.Errorf("Documents section has %d items, want 2", len(filtered[0].items))
	}

	// Subcollections should be filtered
	if len(filtered[1].items) != 1 {
		t.Errorf("Subcollections section has %d items, want 1", len(filtered[1].items))
	}
	if filtered[1].items[0].id != "sub-doc" {
		t.Errorf("Subcollections item = %q, want %q", filtered[1].items[0].id, "sub-doc")
	}

	// Data section should not be filtered
	if len(filtered[2].items) != 1 {
		t.Errorf("Data section has %d items, want 1", len(filtered[2].items))
	}
}

func TestFilterSectionsEmpty(t *testing.T) {
	sections := []Section{
		{title: "Documents", items: []ListItem{{id: "abc"}}},
	}

	filtered := FilterSections(sections, "")
	if len(filtered[0].items) != 1 {
		t.Error("empty pattern should return all items")
	}
}

func TestFilterSectionsNoMatch(t *testing.T) {
	sections := []Section{
		{title: "Documents", items: []ListItem{{id: "abc"}}},
	}

	filtered := FilterSections(sections, "xyz")
	if !filtered[0].hidden {
		t.Error("section with no matches should be hidden")
	}
	if len(filtered[0].items) != 0 {
		t.Errorf("section with no matches should have 0 items, got %d", len(filtered[0].items))
	}
}
