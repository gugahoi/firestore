package browse

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantArgs []string
	}{
		{"", "", nil},
		{"help", "help", nil},
		{"goto users", "goto", []string{"users"}},
		{"sort createdAt desc", "sort", []string{"createdAt", "desc"}},
		{"  goto  users/abc123  ", "goto", []string{"users/abc123"}},
		{"refresh", "refresh", nil},
	}

	for _, tt := range tests {
		name, args := ParseCommand(tt.input)
		if name != tt.wantName {
			t.Errorf("ParseCommand(%q) name = %q, want %q", tt.input, name, tt.wantName)
		}
		if len(args) != len(tt.wantArgs) {
			t.Errorf("ParseCommand(%q) args len = %d, want %d", tt.input, len(args), len(tt.wantArgs))
			continue
		}
		for i := range args {
			if args[i] != tt.wantArgs[i] {
				t.Errorf("ParseCommand(%q) args[%d] = %q, want %q", tt.input, i, args[i], tt.wantArgs[i])
			}
		}
	}
}

func TestCommandRegistryComplete(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Description: "help desc"})
	r.Register(Command{Name: "goto", Description: "goto desc"})
	r.Register(Command{Name: "get", Description: "get desc"})
	r.Register(Command{Name: "sort", Description: "sort desc"})
	r.Register(Command{Name: "refresh", Description: "refresh desc"})

	tests := []struct {
		prefix string
		want   []string
	}{
		{"g", []string{"get", "goto"}},
		{"go", []string{"goto"}},
		{"s", []string{"sort"}},
		{"r", []string{"refresh"}},
		{"h", []string{"help"}},
		{"x", nil},
		{"", []string{"get", "goto", "help", "refresh", "sort"}},
	}

	for _, tt := range tests {
		got := r.Complete(tt.prefix)
		if len(got) != len(tt.want) {
			t.Errorf("Complete(%q) = %v, want %v", tt.prefix, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Complete(%q)[%d] = %q, want %q", tt.prefix, i, got[i], tt.want[i])
			}
		}
	}
}

func TestCommandRegistryGet(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "help", Description: "help desc"})

	cmd, ok := r.Get("help")
	if !ok {
		t.Fatal("expected to find 'help' command")
	}
	if cmd.Name != "help" {
		t.Errorf("cmd.Name = %q, want %q", cmd.Name, "help")
	}

	_, ok = r.Get("unknown")
	if ok {
		t.Error("expected not to find 'unknown' command")
	}
}

func TestCommandRegistryAll(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "zebra"})
	r.Register(Command{Name: "alpha"})
	r.Register(Command{Name: "middle"})

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All() returned %d commands, want 3", len(all))
	}
	if all[0].Name != "alpha" || all[1].Name != "middle" || all[2].Name != "zebra" {
		t.Errorf("All() not sorted: got %v", []string{all[0].Name, all[1].Name, all[2].Name})
	}
}

func TestLongestCommonPrefix(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, ""},
		{[]string{"goto"}, "goto"},
		{[]string{"goto", "get"}, "g"},
		{[]string{"sort", "sort"}, "sort"},
		{[]string{"abc", "xyz"}, ""},
	}

	for _, tt := range tests {
		got := longestCommonPrefix(tt.input)
		if got != tt.want {
			t.Errorf("longestCommonPrefix(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
