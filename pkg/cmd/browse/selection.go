package browse

// Selection tracks a set of selected indices with optional range-select anchor.
type Selection struct {
	indices map[int]bool
	anchor  int
	hasAnchor bool
}

func NewSelection() *Selection {
	return &Selection{
		indices: make(map[int]bool),
	}
}

func (s *Selection) Toggle(index int) {
	if s.indices[index] {
		delete(s.indices, index)
	} else {
		s.indices[index] = true
	}
	s.hasAnchor = false
}

func (s *Selection) SetAnchor(index int) {
	s.hasAnchor = true
	s.anchor = index
	s.indices[index] = true
}

func (s *Selection) ExtendTo(index int) {
	if !s.hasAnchor {
		return
	}
	// Clear previous range, keep only anchor
	s.indices = map[int]bool{s.anchor: true}
	start, end := s.anchor, index
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		s.indices[i] = true
	}
}

func (s *Selection) IsSelected(index int) bool {
	return s.indices[index]
}

func (s *Selection) Count() int {
	return len(s.indices)
}

func (s *Selection) Indices() []int {
	result := make([]int, 0, len(s.indices))
	for i := range s.indices {
		result = append(result, i)
	}
	return result
}

func (s *Selection) Clear() {
	s.indices = make(map[int]bool)
	s.hasAnchor = false
}

func (s *Selection) HasAnchor() bool {
	return s.hasAnchor
}
