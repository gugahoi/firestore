package browse

import (
	"testing"
)

func TestJumplistPushAndBack(t *testing.T) {
	j := NewJumplist()
	j.Push("/users", false)
	j.Push("/users/abc", true)
	j.Push("/orders", false)

	entry, ok := j.Back()
	if !ok {
		t.Fatal("expected back to succeed")
	}
	if entry.Path != "/users/abc" {
		t.Errorf("expected /users/abc, got %s", entry.Path)
	}

	entry, ok = j.Back()
	if !ok {
		t.Fatal("expected back to succeed")
	}
	if entry.Path != "/users" {
		t.Errorf("expected /users, got %s", entry.Path)
	}
}

func TestJumplistForward(t *testing.T) {
	j := NewJumplist()
	j.Push("/a", false)
	j.Push("/b", false)
	j.Push("/c", false)

	j.Back()
	j.Back()

	entry, ok := j.Forward()
	if !ok {
		t.Fatal("expected forward to succeed")
	}
	if entry.Path != "/b" {
		t.Errorf("expected /b, got %s", entry.Path)
	}

	entry, ok = j.Forward()
	if !ok {
		t.Fatal("expected forward to succeed")
	}
	if entry.Path != "/c" {
		t.Errorf("expected /c, got %s", entry.Path)
	}
}

func TestJumplistBoundary(t *testing.T) {
	j := NewJumplist()

	_, ok := j.Back()
	if ok {
		t.Error("expected back on empty jumplist to fail")
	}

	j.Push("/a", false)
	_, ok = j.Back()
	if ok {
		t.Error("expected back at beginning to fail")
	}

	_, ok = j.Forward()
	if ok {
		t.Error("expected forward at end to fail")
	}
}

func TestJumplistTruncation(t *testing.T) {
	j := NewJumplist()
	j.Push("/a", false)
	j.Push("/b", false)
	j.Push("/c", false)

	j.Back()
	j.Back()
	// Now at /a, forward history is /b, /c
	// Push new path should truncate /b, /c
	j.Push("/d", false)

	if j.Len() != 2 {
		t.Errorf("expected 2 entries after truncation, got %d", j.Len())
	}

	_, ok := j.Forward()
	if ok {
		t.Error("expected forward after truncation to fail")
	}
}

func TestJumplistPushIsDoc(t *testing.T) {
	j := NewJumplist()
	j.Push("/users", false)
	j.Push("/users/abc", true)

	entry, _ := j.Back()
	if entry.IsDoc {
		t.Error("expected /users to not be a doc")
	}
}
