package browse

import "testing"

func TestShouldShowPreview(t *testing.T) {
	previewNodes := []ListItem{{path: "users/abc", isDoc: true}}

	tests := []struct {
		name           string
		previewEnabled bool
		previewNodes   []ListItem
		columns        []Column
		activeColumn   int
		want           bool
	}{
		{
			name:           "preview disabled",
			previewEnabled: false,
			previewNodes:   previewNodes,
			columns:        []Column{{path: "users", isDoc: false}},
			activeColumn:   0,
			want:           false,
		},
		{
			name:           "no preview nodes",
			previewEnabled: true,
			previewNodes:   nil,
			columns:        []Column{{path: "users", isDoc: false}},
			activeColumn:   0,
			want:           false,
		},
		{
			name:           "active column is a collection",
			previewEnabled: true,
			previewNodes:   previewNodes,
			columns:        []Column{{path: "users", isDoc: false}},
			activeColumn:   0,
			want:           true,
		},
		{
			name:           "active column is a document",
			previewEnabled: true,
			previewNodes:   previewNodes,
			columns:        []Column{{path: "users", isDoc: false}, {path: "users/abc", isDoc: true}},
			activeColumn:   1,
			want:           false,
		},
		{
			name:           "active column out of range",
			previewEnabled: true,
			previewNodes:   previewNodes,
			columns:        []Column{{path: "users", isDoc: false}},
			activeColumn:   5,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				previewEnabled: tt.previewEnabled,
				previewNodes:   tt.previewNodes,
				columns:        tt.columns,
				activeColumn:   tt.activeColumn,
			}
			if got := m.shouldShowPreview(); got != tt.want {
				t.Errorf("shouldShowPreview() = %v, want %v", got, tt.want)
			}
		})
	}
}
