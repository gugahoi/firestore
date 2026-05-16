package browse

import (
	"sort"
	"testing"
)

func TestSelectionToggle(t *testing.T) {
	s := NewSelection()

	s.Toggle(3)
	if !s.IsSelected(3) {
		t.Error("expected index 3 to be selected")
	}
	if s.Count() != 1 {
		t.Errorf("expected count 1, got %d", s.Count())
	}

	s.Toggle(3)
	if s.IsSelected(3) {
		t.Error("expected index 3 to be deselected")
	}
	if s.Count() != 0 {
		t.Errorf("expected count 0, got %d", s.Count())
	}
}

func TestSelectionRangeSelect(t *testing.T) {
	s := NewSelection()

	s.SetAnchor(2)
	s.ExtendTo(5)

	expected := []int{2, 3, 4, 5}
	if s.Count() != len(expected) {
		t.Errorf("expected count %d, got %d", len(expected), s.Count())
	}
	for _, i := range expected {
		if !s.IsSelected(i) {
			t.Errorf("expected index %d to be selected", i)
		}
	}
}

func TestSelectionRangeSelectReverse(t *testing.T) {
	s := NewSelection()

	s.SetAnchor(5)
	s.ExtendTo(2)

	expected := []int{2, 3, 4, 5}
	if s.Count() != len(expected) {
		t.Errorf("expected count %d, got %d", len(expected), s.Count())
	}
	for _, i := range expected {
		if !s.IsSelected(i) {
			t.Errorf("expected index %d to be selected", i)
		}
	}
}

func TestSelectionRangeExtendUpdates(t *testing.T) {
	s := NewSelection()

	s.SetAnchor(3)
	s.ExtendTo(6)
	if s.Count() != 4 {
		t.Errorf("expected count 4, got %d", s.Count())
	}

	// Shrink range
	s.ExtendTo(4)
	if s.Count() != 2 {
		t.Errorf("expected count 2, got %d", s.Count())
	}
	if !s.IsSelected(3) || !s.IsSelected(4) {
		t.Error("expected indices 3 and 4 to be selected")
	}
	if s.IsSelected(5) || s.IsSelected(6) {
		t.Error("expected indices 5 and 6 to be deselected")
	}
}

func TestSelectionClear(t *testing.T) {
	s := NewSelection()
	s.Toggle(1)
	s.Toggle(2)
	s.Toggle(3)

	s.Clear()
	if s.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", s.Count())
	}
	if s.HasAnchor() {
		t.Error("expected no anchor after clear")
	}
}

func TestSelectionIndices(t *testing.T) {
	s := NewSelection()
	s.Toggle(5)
	s.Toggle(1)
	s.Toggle(3)

	indices := s.Indices()
	sort.Ints(indices)

	expected := []int{1, 3, 5}
	if len(indices) != len(expected) {
		t.Fatalf("expected %d indices, got %d", len(expected), len(indices))
	}
	for i := range indices {
		if indices[i] != expected[i] {
			t.Errorf("indices[%d] = %d, want %d", i, indices[i], expected[i])
		}
	}
}
