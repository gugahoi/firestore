package browse

import (
	"strings"
)

// getEffectiveSections returns the sections for display, applying any active filter.
func (m *Model) getEffectiveSections(col Column) []Section {
	if m.filterPattern == "" {
		return col.sections
	}
	// Only filter the active column
	if m.activeColumn < len(m.columns) && &m.columns[m.activeColumn] != nil && col.path == m.columns[m.activeColumn].path {
		return FilterSections(col.sections, m.filterPattern)
	}
	return col.sections
}

// FilterItems returns only items whose display ID contains the pattern (case-insensitive).
// Returns all items if pattern is empty.
func FilterItems(items []ListItem, pattern string) []ListItem {
	if pattern == "" {
		return items
	}
	lower := strings.ToLower(pattern)
	var result []ListItem
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.id), lower) {
			result = append(result, item)
		}
	}
	return result
}

// FilterSections returns sections with items filtered by pattern.
// Only non-data sections (Documents, Subcollections) are filtered.
// Data sections are returned as-is since they represent document fields.
func FilterSections(sections []Section, pattern string) []Section {
	if pattern == "" {
		return sections
	}
	var result []Section
	for _, s := range sections {
		if s.title == "Data" || s.title == "Metadata" {
			result = append(result, s)
			continue
		}
		filtered := FilterItems(s.items, pattern)
		result = append(result, Section{
			title:  s.title,
			items:  filtered,
			hidden: len(filtered) == 0,
		})
	}
	return result
}
