package browse

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/gugahoi/firestore/pkg/cmd/document"
)

// These tests run only against a Firestore emulator. Set FIRESTORE_EMULATOR_HOST
// (e.g. localhost:8765) before running them.
func emulatorClient(t *testing.T) *firestore.Client {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set; skipping emulator integration test")
	}
	client, err := firestore.NewClient(context.Background(), "test-project")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestExecuteRename_SimpleRename(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "users/alice"
	dst := "users/alice2"
	data := map[string]interface{}{"name": "Alice", "age": int64(30)}
	if _, err := client.Doc(src).Set(ctx, data); err != nil {
		t.Fatalf("seed: %v", err)
	}

	msg := executeRename(client, src, dst, false)()
	if _, ok := msg.(renamedMsg); !ok {
		t.Fatalf("expected renamedMsg, got %#v", msg)
	}

	// Source gone.
	if _, err := client.Doc(src).Get(ctx); err == nil {
		t.Errorf("source %s still exists after rename", src)
	}
	// Destination has the same data.
	snap, err := client.Doc(dst).Get(ctx)
	if err != nil {
		t.Fatalf("destination %s missing: %v", dst, err)
	}
	if got := snap.Data()["name"]; got != "Alice" {
		t.Errorf("destination name = %v, want Alice", got)
	}
}

func TestExecuteRename_CrossCollectionMove(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "users/bob"
	dst := "archive/bob"
	if _, err := client.Doc(src).Set(ctx, map[string]interface{}{"name": "Bob"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, ok := executeRename(client, src, dst, true)().(renamedMsg); !ok {
		t.Fatal("expected renamedMsg")
	}

	if _, err := client.Doc(src).Get(ctx); err == nil {
		t.Errorf("source %s still exists", src)
	}
	if _, err := client.Doc(dst).Get(ctx); err != nil {
		t.Errorf("destination %s missing: %v", dst, err)
	}
}

func TestPrepareRename_DetectsExistingDestination(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "users/carol"
	dst := "users/dave"
	if _, err := client.Doc(src).Set(ctx, map[string]interface{}{"v": 1}); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	if _, err := client.Doc(dst).Set(ctx, map[string]interface{}{"v": 2}); err != nil {
		t.Fatalf("seed dst: %v", err)
	}

	msg := prepareRename(client, src, dst, false)()
	prepared, ok := msg.(renamePreparedMsg)
	if !ok {
		t.Fatalf("expected renamePreparedMsg, got %#v", msg)
	}
	if !prepared.destExists {
		t.Error("expected destExists=true for existing destination")
	}

	// Free destination reports destExists=false.
	msg2 := prepareRename(client, src, "users/free-slot", false)()
	prepared2, ok := msg2.(renamePreparedMsg)
	if !ok {
		t.Fatalf("expected renamePreparedMsg, got %#v", msg2)
	}
	if prepared2.destExists {
		t.Error("expected destExists=false for free destination")
	}
}

func TestPrepareRename_RefusesSubcollections(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	src := "users/eve"
	if _, err := client.Doc(src).Set(ctx, map[string]interface{}{"name": "Eve"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := client.Doc(src + "/orders/o1").Set(ctx, map[string]interface{}{"total": 10}); err != nil {
		t.Fatalf("seed subcollection: %v", err)
	}

	msg := prepareRename(client, src, "users/eve2", false)()
	if _, ok := msg.(renameRefusedMsg); !ok {
		t.Fatalf("expected renameRefusedMsg for doc with subcollections, got %#v", msg)
	}

	// HasSubcollections directly.
	has, err := document.HasSubcollections(ctx, client.Doc(src))
	if err != nil {
		t.Fatalf("HasSubcollections: %v", err)
	}
	if !has {
		t.Error("HasSubcollections = false, want true")
	}
}
