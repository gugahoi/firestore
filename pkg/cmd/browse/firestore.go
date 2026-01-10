package browse

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"cloud.google.com/go/firestore"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/iterator"
)

// fetchColumnData fetches data for a specific column based on path and type
func fetchColumnData(client *firestore.Client, path string, isDoc bool, columnIndex int) tea.Cmd {
	return func() tea.Msg {
		logDebug("fetchColumnData started: path='%s', isDoc=%v, columnIndex=%d", path, isDoc, columnIndex)
		ctx := context.Background()
		sections := []Section{}
		docContent := ""

		if path == "" {
			// Root: fetch root collections
			iter := client.Collections(ctx)
			var items []ListItem
			for {
				colRef, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return errorMsg{err: err}
				}
				// Use ID as path since root collections don't have parent paths
				items = append(items, ListItem{
					id:    colRef.ID,
					path:  colRef.ID,
					isDoc: false,
				})
			}
			sections = append(sections, Section{
				title:  "Collections",
				items:  items,
				hidden: len(items) == 0,
			})
		} else if !isDoc {
			// Collection: fetch documents
			colPath := strings.TrimPrefix(path, "/")
			colRef := client.Collection(colPath)
			iter := colRef.Documents(ctx)
			var items []ListItem
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return errorMsg{err: err}
				}
				metadata := map[string]string{}
				if !doc.CreateTime.IsZero() {
					metadata["created"] = doc.CreateTime.Local().Format("2006-01-02 15:04:05")
				}
				// Build relative path: parent_path/doc_id
				docPath := path + "/" + doc.Ref.ID
				items = append(items, ListItem{
					id:       doc.Ref.ID,
					path:     docPath,
					isDoc:    true,
					metadata: metadata,
				})
			}
			sections = append(sections, Section{
				title:  "Documents",
				items:  items,
				hidden: len(items) == 0,
			})
		} else {
			// Document: fetch subcollections AND document content
			docPath := strings.TrimPrefix(path, "/")
			docRef := client.Doc(docPath)

			// Fetch subcollections
			iter := docRef.Collections(ctx)
			var items []ListItem
			for {
				colRef, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return errorMsg{err: err}
				}
				// Build relative path: parent_path/subcollection_id
				subcolPath := path + "/" + colRef.ID
				items = append(items, ListItem{
					id:    colRef.ID,
					path:  subcolPath,
					isDoc: false,
				})
			}
			if len(items) > 0 {
				sections = append(sections, Section{
					title:  "Subcollections",
					items:  items,
					hidden: false,
				})
			}

			// Fetch document data
			snap, err := docRef.Get(ctx)
			if err != nil {
				return errorMsg{err: err}
			}

			// Format as JSON (following pattern from document/get.go)
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(snap.Data()); err != nil {
				return errorMsg{err: err}
			}
			docContent = buf.String()
		}

		result := fetchedColumnMsg{
			columnIndex: columnIndex,
			sections:    sections,
			docContent:  docContent,
		}
		logDebug("fetchColumnData completed: columnIndex=%d, sections=%d, docContentLen=%d",
			columnIndex, len(sections), len(docContent))
		return result
	}
}
