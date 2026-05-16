package browse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
)

// ExportJSON serializes documents as a pretty-printed JSON array.
func ExportJSON(docs []map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(docs); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// ExportNDJSON serializes documents as newline-delimited JSON (one object per line).
func ExportNDJSON(docs []map[string]interface{}) (string, error) {
	var lines []string
	for _, doc := range docs {
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(doc); err != nil {
			return "", err
		}
		lines = append(lines, strings.TrimSpace(buf.String()))
	}
	return strings.Join(lines, "\n"), nil
}

func cmdExport(m *Model, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("usage: :export json|ndjson [filename]")
	}

	format := strings.ToLower(args[0])
	if format != "json" && format != "ndjson" {
		return "", fmt.Errorf("unknown format %q, use json or ndjson", format)
	}

	filename := ""
	if len(args) > 1 {
		filename = args[1]
	}

	// Gather documents to export
	docs, err := gatherExportDocs(m)
	if err != nil {
		return "", err
	}

	if len(docs) == 0 {
		return "", fmt.Errorf("no documents to export")
	}

	// Serialize
	var content string
	switch format {
	case "json":
		content, err = ExportJSON(docs)
	case "ndjson":
		content, err = ExportNDJSON(docs)
	}
	if err != nil {
		return "", fmt.Errorf("serialization error: %w", err)
	}

	// Write to file or clipboard
	if filename != "" {
		if err := os.WriteFile(filename, []byte(content+"\n"), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		return fmt.Sprintf("Exported %d documents to %s", len(docs), filename), nil
	}

	if err := clipboard.WriteAll(content); err != nil {
		return "", fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return fmt.Sprintf("Exported %d documents to clipboard", len(docs)), nil
}

func gatherExportDocs(m *Model) ([]map[string]interface{}, error) {
	if m.activeColumn >= len(m.columns) {
		return nil, fmt.Errorf("no active column")
	}

	col := m.columns[m.activeColumn]

	// Single document view
	if col.isDoc && col.docData != nil {
		return []map[string]interface{}{col.docData}, nil
	}

	// Collection view with visual selection
	if m.selection.Count() > 0 {
		return gatherSelectedDocs(m, col)
	}

	// All loaded documents in collection
	return gatherAllDocs(m, col)
}

func gatherSelectedDocs(m *Model, col Column) ([]map[string]interface{}, error) {
	sections := m.getEffectiveSections(col)
	var paths []string
	itemIndex := 0
	for _, section := range sections {
		if section.hidden {
			continue
		}
		for _, item := range section.items {
			if m.selection.IsSelected(itemIndex) && item.isDoc && item.path != "__load_more__" {
				paths = append(paths, item.path)
			}
			itemIndex++
		}
	}

	return fetchDocData(m, paths)
}

func gatherAllDocs(m *Model, col Column) ([]map[string]interface{}, error) {
	sections := m.getEffectiveSections(col)
	var paths []string
	for _, section := range sections {
		if section.hidden {
			continue
		}
		for _, item := range section.items {
			if item.isDoc && item.path != "__load_more__" {
				paths = append(paths, item.path)
			}
		}
	}

	return fetchDocData(m, paths)
}

func fetchDocData(m *Model, paths []string) ([]map[string]interface{}, error) {
	var docs []map[string]interface{}
	for _, path := range paths {
		docRef := m.client.Doc(strings.TrimPrefix(path, "/"))
		snap, err := docRef.Get(m.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", path, err)
		}
		if snap.Exists() {
			docs = append(docs, snap.Data())
		}
	}
	return docs, nil
}
