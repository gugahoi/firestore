package browse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/firestore"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Helper to build tree nodes from map
func buildTreeNodes(data map[string]interface{}) []ListItem {
	var nodes []ListItem

	// Sort keys for consistent order
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		nodes = append(nodes, createNode(k, v, 0))
	}

	return nodes
}

func createNode(key string, value interface{}, depth int) ListItem {
	item := ListItem{
		id:       key,
		isData:   true,
		key:      key,
		depth:    depth,
		expanded: false, // Default to collapsed
	}

	switch v := value.(type) {
	case map[string]interface{}:
		item.dataType = "object"
		item.valueStr = "{...}"

		// Create children
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			item.children = append(item.children, createNode(k, v[k], depth+1))
		}

	case []interface{}:
		item.dataType = "array"
		item.valueStr = fmt.Sprintf("[%d]", len(v))

		for i, val := range v {
			item.children = append(item.children, createNode(fmt.Sprintf("%d", i), val, depth+1))
		}

	case string:
		item.dataType = "string"
		item.valueStr = v // Don't quote it here, handle quotes in view.go

	case float64:
		item.dataType = "number"
		item.valueStr = fmt.Sprintf("%v", v)

	case bool:
		item.dataType = "bool"
		item.valueStr = fmt.Sprintf("%v", v)

	case nil:
		item.dataType = "null"
		item.valueStr = "null"

	default:
		// Handle other types (e.g. timestamps)
		item.dataType = "other"
		item.valueStr = fmt.Sprintf("%v", v)
	}

	return item
}

type fetchOpts struct {
	limit       int
	offset      int
	appendItems bool
}

type fetchOption func(*fetchOpts)

func withLimit(n int) fetchOption {
	return func(o *fetchOpts) { o.limit = n }
}

func withOffset(n int) fetchOption {
	return func(o *fetchOpts) { o.offset = n }
}

func withAppend() fetchOption {
	return func(o *fetchOpts) { o.appendItems = true }
}

func deleteDocument(client *firestore.Client, path string, fromDocView bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		docRef := client.Doc(strings.TrimPrefix(path, "/"))
		_, err := docRef.Delete(ctx)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to delete document: %w", err)}
		}
		return documentDeletedMsg{path: path, fromDocView: fromDocView}
	}
}

// fetchColumnData fetches data for a specific column based on path and type
func fetchColumnData(client *firestore.Client, path string, isDoc bool, columnIndex int, sortField string, sortDirection firestore.Direction, opts ...fetchOption) tea.Cmd {
	var fo fetchOpts
	for _, o := range opts {
		o(&fo)
	}
	return func() tea.Msg {
		logDebug("fetchColumnData started: path='%s', isDoc=%v, columnIndex=%d, sortField='%s', limit=%d, offset=%d", path, isDoc, columnIndex, sortField, fo.limit, fo.offset)
		ctx := context.Background()
		sections := []Section{}
		docContent := ""
		var docData map[string]interface{}
		docMetadata := map[string]string{}
		availableFields := []string{}
		hasMore := false

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

			// Apply sorting if specified
			query := colRef.Query
			if sortField != "" {
				query = query.OrderBy(sortField, sortDirection)
			}

			// Apply pagination limit
			if fo.limit > 0 {
				query = query.Limit(fo.limit + 1) // Fetch one extra to detect hasMore
				if fo.offset > 0 {
					query = query.Offset(fo.offset)
				}
			}

			iter := query.Documents(ctx)
			var items []ListItem
			firstDoc := true
			fetchCount := 0
			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return errorMsg{err: err}
				}

				fetchCount++
				// If we got more than limit, we have more pages
				if fo.limit > 0 && fetchCount > fo.limit {
					hasMore = true
					break
				}

				// Extract field names from first document
				if firstDoc {
					firstDoc = false
					data := doc.Data()
					for key := range data {
						availableFields = append(availableFields, key)
					}
					sort.Strings(availableFields)
				}

				metadata := map[string]string{}
				if !doc.CreateTime.IsZero() {
					metadata["created"] = doc.CreateTime.Local().Format("2006-01-02 15:04:05")
				}
				docPath := path + "/" + doc.Ref.ID
				items = append(items, ListItem{
					id:       doc.Ref.ID,
					path:     docPath,
					isDoc:    true,
					metadata: metadata,
				})
			}

			// Only fetch document refs for missing docs on first page
			if fo.offset == 0 {
				seen := make(map[string]bool, len(items))
				for _, item := range items {
					seen[item.id] = true
				}

				refsIter := colRef.DocumentRefs(ctx)
				for {
					ref, err := refsIter.Next()
					if err == iterator.Done {
						break
					}
					if err != nil {
						return errorMsg{err: err}
					}
					if seen[ref.ID] {
						continue
					}
					docPath := path + "/" + ref.ID
					items = append(items, ListItem{
						id:        ref.ID,
						path:      docPath,
						isDoc:     true,
						isMissing: true,
					})
				}
			}

			// Add "Load more..." sentinel if there are more pages
			if hasMore {
				items = append(items, ListItem{
					id:   "Load more...",
					path: "__load_more__",
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
			if err != nil && status.Code(err) != codes.NotFound {
				return errorMsg{err: err}
			}

			if snap != nil && snap.Exists() {
				// Extract metadata
				docMetadata = map[string]string{}
				if !snap.CreateTime.IsZero() {
					docMetadata["Created"] = snap.CreateTime.Local().Format("2006-01-02 15:04:05")
				}
				if !snap.UpdateTime.IsZero() {
					docMetadata["Updated"] = snap.UpdateTime.Local().Format("2006-01-02 15:04:05")
				}
				if !snap.ReadTime.IsZero() {
					docMetadata["Read"] = snap.ReadTime.Local().Format("2006-01-02 15:04:05")
				}

				// Format as JSON
				docData = snap.Data()
				var buf bytes.Buffer
				encoder := json.NewEncoder(&buf)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(docData); err != nil {
					return errorMsg{err: err}
				}
				docContent = buf.String()

				// Build tree nodes for structured view
				rootNodes := buildTreeNodes(docData)
				if len(rootNodes) > 0 {
					sections = append(sections, Section{
						title:  "Data",
						items:  rootNodes,
						hidden: false,
					})
				}
			}

			// Create Metadata section
			if len(docMetadata) > 0 {
				var metadataItems []ListItem
				// Sort keys for consistent order
				keys := []string{"Created", "Updated", "Read"}
				for _, k := range keys {
					if v, ok := docMetadata[k]; ok {
						metadataItems = append(metadataItems, ListItem{
							id:     fmt.Sprintf("%s: %s", k, v),
							isData: false,
							isDoc:  false,
						})
					}
				}
				sections = append(sections, Section{
					title:  "Metadata",
					items:  metadataItems,
					hidden: false,
				})
			}
		}

		// Count documents in sections
		totalDocs := 0
		for _, s := range sections {
			if s.title == "Documents" {
				for _, item := range s.items {
					if item.path != "__load_more__" {
						totalDocs++
					}
				}
			}
		}

		result := fetchedColumnMsg{
			columnIndex:     columnIndex,
			sections:        sections,
			docContent:      docContent,
			docData:         docData,
			docMetadata:     docMetadata,
			availableFields: availableFields,
			hasMore:         hasMore,
			docCount:        totalDocs,
			appendItems:     fo.appendItems,
		}
		logDebug("fetchColumnData completed: columnIndex=%d, sections=%d, docContentLen=%d, docData keys=%d",
			columnIndex, len(sections), len(docContent), len(docData))
		return result
	}
}
