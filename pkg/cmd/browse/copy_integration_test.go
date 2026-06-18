//go:build integration

package browse

import (
	"context"
	"testing"

	"github.com/gugahoi/firestore/pkg/cmd/document"
)

func TestExecuteCopy_AutoID(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "copy_users/alice"
	data := map[string]interface{}{"name": "Alice", "age": int64(30)}
	if _, err := client.Doc(src).Set(ctx, data); err != nil {
		t.Fatalf("seed: %v", err)
	}

	msg := executeCopy(client, src, "", false)()
	copied, ok := msg.(copiedMsg)
	if !ok {
		t.Fatalf("expected copiedMsg, got %#v", msg)
	}
	if copied.newPath == src {
		t.Fatalf("auto copy reused source path %s", src)
	}
	if copied.hadSubcollections {
		t.Error("hadSubcollections = true, want false")
	}

	// Source untouched.
	if _, err := client.Doc(src).Get(ctx); err != nil {
		t.Errorf("source %s missing after copy: %v", src, err)
	}
	// New document has the same data.
	snap, err := client.Doc(copied.newPath).Get(ctx)
	if err != nil {
		t.Fatalf("copy %s missing: %v", copied.newPath, err)
	}
	if got := snap.Data()["name"]; got != "Alice" {
		t.Errorf("copy name = %v, want Alice", got)
	}
}

func TestExecuteCopy_ExplicitID(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "copy_users/bob"
	dst := "copy_users/bob-copy"
	if _, err := client.Doc(src).Set(ctx, map[string]interface{}{"name": "Bob"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	msg := executeCopy(client, src, dst, false)()
	copied, ok := msg.(copiedMsg)
	if !ok {
		t.Fatalf("expected copiedMsg, got %#v", msg)
	}
	if copied.newPath != dst {
		t.Errorf("newPath = %q, want %q", copied.newPath, dst)
	}

	// Both source and copy exist with the same data.
	if _, err := client.Doc(src).Get(ctx); err != nil {
		t.Errorf("source %s missing: %v", src, err)
	}
	snap, err := client.Doc(dst).Get(ctx)
	if err != nil {
		t.Fatalf("copy %s missing: %v", dst, err)
	}
	if got := snap.Data()["name"]; got != "Bob" {
		t.Errorf("copy name = %v, want Bob", got)
	}
}

func TestExecuteCopy_RefusesExistingDestination(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "copy_users/carol"
	dst := "copy_users/dave"
	if _, err := client.Doc(src).Set(ctx, map[string]interface{}{"v": int64(1)}); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	if _, err := client.Doc(dst).Set(ctx, map[string]interface{}{"v": int64(2)}); err != nil {
		t.Fatalf("seed dst: %v", err)
	}

	msg := executeCopy(client, src, dst, false)()
	if _, ok := msg.(copyRefusedMsg); !ok {
		t.Fatalf("expected copyRefusedMsg for existing destination, got %#v", msg)
	}

	// Destination is unchanged (not overwritten).
	snap, err := client.Doc(dst).Get(ctx)
	if err != nil {
		t.Fatalf("destination %s missing: %v", dst, err)
	}
	if got := snap.Data()["v"]; got != int64(2) {
		t.Errorf("destination v = %v, want 2 (unchanged)", got)
	}
}

func TestExecuteCopy_PhantomDocument(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	// A "phantom" document: no own fields, only a subcollection. These are
	// listed as first-class refs in the browser, so a user can select one and
	// press `c`. Probe what executeCopy does with it.
	src := "copy_users/ghost"
	if _, err := client.Doc(src + "/orders/o1").Set(ctx, map[string]interface{}{"total": int64(5)}); err != nil {
		t.Fatalf("seed subcollection: %v", err)
	}

	// A phantom source has no field data, so copy is refused with a transient
	// message rather than a sticky error.
	msg := executeCopy(client, src, "", false)()
	if _, ok := msg.(copyRefusedMsg); !ok {
		t.Fatalf("expected copyRefusedMsg for phantom document, got %#v", msg)
	}
}

func TestExecuteCopy_IgnoresSubcollections(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "copy_users/eve"
	dst := "copy_users/eve-copy"
	if _, err := client.Doc(src).Set(ctx, map[string]interface{}{"name": "Eve"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := client.Doc(src + "/orders/o1").Set(ctx, map[string]interface{}{"total": int64(10)}); err != nil {
		t.Fatalf("seed subcollection: %v", err)
	}

	msg := executeCopy(client, src, dst, false)()
	copied, ok := msg.(copiedMsg)
	if !ok {
		t.Fatalf("expected copiedMsg, got %#v", msg)
	}
	if !copied.hadSubcollections {
		t.Error("hadSubcollections = false, want true")
	}

	// Field data copied.
	snap, err := client.Doc(dst).Get(ctx)
	if err != nil {
		t.Fatalf("copy %s missing: %v", dst, err)
	}
	if got := snap.Data()["name"]; got != "Eve" {
		t.Errorf("copy name = %v, want Eve", got)
	}
	// Subcollection NOT copied.
	has, err := document.HasSubcollections(ctx, client.Doc(dst))
	if err != nil {
		t.Fatalf("check copy subcollections: %v", err)
	}
	if has {
		t.Error("copy has subcollections, want none copied")
	}
}
