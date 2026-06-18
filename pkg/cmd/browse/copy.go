package browse

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gugahoi/firestore/pkg/cmd/document"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// copiedMsg is emitted once the source document's field data has been copied
// into a new document. newPath is the path of the freshly created document,
// collectionPath is the collection the new sibling now lives in (captured at
// copy time so the handler can refresh the correct column even if the user
// navigated during the async copy), and hadSubcollections reports whether the
// source had subcollections (which are not copied) so the caller can surface a
// note.
type copiedMsg struct {
	newPath           string
	collectionPath    string
	hadSubcollections bool
}

// copyRefusedMsg is emitted when a copy is rejected because the explicit
// destination already exists. It surfaces as a transient status message rather
// than a sticky error.
type copyRefusedMsg struct {
	reason string
}

// executeCopy copies the source document's field data into a new document. When
// dst is empty an auto-generated id is allocated in the source's collection;
// otherwise dst is used and the copy is refused if a document already exists
// there. Only top-level field data is copied — subcollections are left behind.
// The source is never modified.
func executeCopy(client *firestore.Client, src, dst string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		srcRef := client.Doc(strings.TrimPrefix(src, "/"))

		snap, err := srcRef.Get(ctx)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				// A subcollection-only "phantom" ref has no field data of its
				// own — there is nothing to copy.
				return copyRefusedMsg{reason: "cannot copy: document has no data (subcollection-only)"}
			}
			return errorMsg{err: fmt.Errorf("failed to read source document: %w", err)}
		}

		// Subcollections are not copied; note their presence for the status line.
		hadSub, err := document.HasSubcollections(ctx, srcRef)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to check subcollections: %w", err)}
		}

		if dst == "" {
			// Auto-generated id within the source document's collection.
			parent := parentCollectionPath(src)
			if parent == "" {
				return errorMsg{err: fmt.Errorf("invalid source path: %s", src)}
			}
			dstRef := client.Collection(parent).NewDoc()
			if _, err := dstRef.Set(ctx, snap.Data()); err != nil {
				return errorMsg{err: fmt.Errorf("failed to write copy: %w", err)}
			}
			newPath := parent + "/" + dstRef.ID
			return copiedMsg{newPath: newPath, collectionPath: parentCollectionPath(newPath), hadSubcollections: hadSub}
		}

		// Explicit destination — refuse if something is already there.
		dstRef := client.Doc(strings.TrimPrefix(dst, "/"))
		if _, err := dstRef.Create(ctx, snap.Data()); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				return copyRefusedMsg{reason: fmt.Sprintf("cannot copy: %s already exists", dst)}
			}
			return errorMsg{err: fmt.Errorf("failed to write copy: %w", err)}
		}
		newPath := strings.Trim(dst, "/")
		return copiedMsg{newPath: newPath, collectionPath: parentCollectionPath(newPath), hadSubcollections: hadSub}
	}
}

// parentCollectionPath returns the collection path containing the given
// document path (everything before the final segment). It returns "" if the
// path does not contain a collection segment.
func parentCollectionPath(docPath string) string {
	p := strings.Trim(docPath, "/")
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return ""
	}
	return p[:idx]
}

// copyRefreshIndex returns the index of the collection column that should be
// refreshed after a copy completes, identified by the collection path captured
// in copiedMsg at copy-initiation time. It deliberately does NOT consult
// m.activeColumn: the user may have navigated during the async copy, so the
// live active column can no longer be trusted to point at the originating
// collection. Returns -1 when no current column matches (e.g. the column was
// removed via navigation), in which case nothing should be refreshed.
func (m Model) copyRefreshIndex(collectionPath string) int {
	if collectionPath == "" {
		return -1
	}
	for idx, col := range m.columns {
		if col.isDoc {
			continue
		}
		if strings.TrimPrefix(col.path, "/") == collectionPath {
			return idx
		}
	}
	return -1
}
