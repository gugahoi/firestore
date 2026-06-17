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

// renamePreparedMsg is emitted once the pre-flight checks (subcollections,
// destination existence) have completed successfully.
type renamePreparedMsg struct {
	src         string
	dst         string
	destExists  bool
	fromDocView bool
}

// renamedMsg is emitted once the document has been copied to its destination
// and the source has been deleted.
type renamedMsg struct {
	src         string
	dst         string
	fromDocView bool
}

// renameRefusedMsg is emitted when a rename is rejected by a pre-flight guard
// (e.g. the source has subcollections). It surfaces as a transient status
// message rather than a sticky error.
type renameRefusedMsg struct {
	reason string
}

// resolveRenameTarget computes the destination path for a rename given the
// source document path and the user's input. A bare name (no "/") is a sibling
// rename within the same collection; an input containing "/" is treated as an
// absolute document path from the root.
func resolveRenameTarget(source, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("destination required")
	}

	src := strings.Trim(source, "/")

	var dst string
	if strings.Contains(input, "/") {
		dst = strings.Trim(input, "/")
		segments := strings.Split(dst, "/")
		if len(segments)%2 != 0 {
			return "", fmt.Errorf("destination must be a document path (even number of segments): %s", dst)
		}
		for _, s := range segments {
			if s == "" {
				return "", fmt.Errorf("destination has an empty path segment: %s", dst)
			}
		}
	} else {
		idx := strings.LastIndex(src, "/")
		if idx < 0 {
			return "", fmt.Errorf("invalid source path: %s", source)
		}
		dst = src[:idx+1] + input
	}

	if dst == src {
		return "", fmt.Errorf("source and destination are the same")
	}
	return dst, nil
}

// prepareRename runs the pre-flight checks: it refuses to rename a document
// that has subcollections (a phantom subcollection-only source is caught here
// too) and reports whether the destination already exists so the caller can
// confirm an overwrite.
func prepareRename(client *firestore.Client, src, dst string, fromDocView bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		srcRef := client.Doc(strings.TrimPrefix(src, "/"))

		hasSub, err := document.HasSubcollections(ctx, srcRef)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to check subcollections: %w", err)}
		}
		if hasSub {
			return renameRefusedMsg{reason: "cannot rename: document has subcollections (not yet supported)"}
		}

		dstRef := client.Doc(strings.TrimPrefix(dst, "/"))
		snap, err := dstRef.Get(ctx)
		destExists := false
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return errorMsg{err: fmt.Errorf("failed to check destination: %w", err)}
			}
		} else {
			destExists = snap.Exists()
		}

		return renamePreparedMsg{src: src, dst: dst, destExists: destExists, fromDocView: fromDocView}
	}
}

// executeRename copies the source document to the destination, then deletes the
// source. The copy happens first: if it fails, the source is left untouched and
// nothing is deleted.
func executeRename(client *firestore.Client, src, dst string, fromDocView bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		srcRef := client.Doc(strings.TrimPrefix(src, "/"))

		snap, err := srcRef.Get(ctx)
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to read source document: %w", err)}
		}

		dstRef := client.Doc(strings.TrimPrefix(dst, "/"))
		if _, err := dstRef.Set(ctx, snap.Data()); err != nil {
			return errorMsg{err: fmt.Errorf("failed to write destination: %w", err)}
		}

		if _, err := srcRef.Delete(ctx); err != nil {
			return errorMsg{err: fmt.Errorf("copied to %s but failed to delete source %s: %w", dst, src, err)}
		}

		return renamedMsg{src: src, dst: dst, fromDocView: fromDocView}
	}
}
