package browse

import "testing"

// TestCopyRefreshIndex verifies that the post-copy refresh target is resolved
// from the collection path captured at copy time, NOT from the live
// activeColumn. This is the regression guard for the race where the user
// navigates during an in-flight async copy: by the time copiedMsg arrives,
// activeColumn no longer points at the originating collection.
func TestCopyRefreshIndex(t *testing.T) {
	tests := []struct {
		name           string
		columns        []Column
		activeColumn   int
		collectionPath string
		want           int
	}{
		{
			name: "origin collection found regardless of active column",
			columns: []Column{
				{path: "users", isDoc: false},
				{path: "users/alice", isDoc: true},
				{path: "users/alice/orders", isDoc: false},
			},
			// User navigated two columns to the right during the copy.
			activeColumn:   2,
			collectionPath: "users",
			want:           0,
		},
		{
			name: "origin column removed by navigation -> no refresh",
			columns: []Column{
				{path: "products", isDoc: false},
			},
			// The originating "users" collection is no longer on screen.
			activeColumn:   0,
			collectionPath: "users",
			want:           -1,
		},
		{
			name: "matches collection column, not same-path document column",
			columns: []Column{
				{path: "users", isDoc: false},
				{path: "users/alice", isDoc: true},
			},
			activeColumn:   1,
			collectionPath: "users",
			want:           0,
		},
		{
			name: "tolerates leading slash on column path",
			columns: []Column{
				{path: "/users", isDoc: false},
			},
			activeColumn:   0,
			collectionPath: "users",
			want:           0,
		},
		{
			name: "empty collection path -> no refresh",
			columns: []Column{
				{path: "users", isDoc: false},
			},
			activeColumn:   0,
			collectionPath: "",
			want:           -1,
		},
		{
			name: "nested subcollection origin",
			columns: []Column{
				{path: "users", isDoc: false},
				{path: "users/alice", isDoc: true},
				{path: "users/alice/orders", isDoc: false},
			},
			// Copy of a doc inside the orders subcollection; active column has
			// since moved back to the root.
			activeColumn:   0,
			collectionPath: "users/alice/orders",
			want:           2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				columns:      tt.columns,
				activeColumn: tt.activeColumn,
			}
			if got := m.copyRefreshIndex(tt.collectionPath); got != tt.want {
				t.Errorf("copyRefreshIndex(%q) = %d, want %d", tt.collectionPath, got, tt.want)
			}
		})
	}
}
