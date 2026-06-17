package browse

import "testing"

func TestResolveRenameTarget(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:   "bare name renames within same collection",
			source: "users/abc",
			input:  "xyz",
			want:   "users/xyz",
		},
		{
			name:   "bare name in nested collection",
			source: "users/abc/orders/o1",
			input:  "o2",
			want:   "users/abc/orders/o2",
		},
		{
			name:   "slash path is absolute from root",
			source: "users/abc",
			input:  "archive/abc",
			want:   "archive/abc",
		},
		{
			name:   "leading slash is trimmed",
			source: "users/abc",
			input:  "/archive/abc",
			want:   "archive/abc",
		},
		{
			name:   "source leading slash is handled",
			source: "/users/abc",
			input:  "xyz",
			want:   "users/xyz",
		},
		{
			name:   "bare name without slash stays a sibling rename",
			source: "users/abc",
			input:  "archive",
			want:   "users/archive",
		},
		{
			name:    "odd-segment slash path is rejected",
			source:  "users/abc",
			input:   "archive/old/extra",
			wantErr: true,
		},
		{
			name:    "empty input is rejected",
			source:  "users/abc",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "empty interior segment is rejected",
			source:  "users/abc",
			input:   "a//b/c",
			wantErr: true,
		},
		{
			name:    "same destination is rejected (bare)",
			source:  "users/abc",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "same destination is rejected (absolute)",
			source:  "users/abc",
			input:   "users/abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveRenameTarget(tt.source, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveRenameTarget(%q, %q) = %q, want %q", tt.source, tt.input, got, tt.want)
			}
		})
	}
}
