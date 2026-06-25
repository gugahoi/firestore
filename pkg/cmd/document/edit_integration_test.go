//go:build integration

package document

import (
	"context"
	"os"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
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

func TestEditFields_PartialMerge(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	path := "edit_users/alice"
	seed := map[string]any{
		"name":  "Alice",
		"age":   int64(30),
		"email": "alice@example.com",
	}
	if _, err := client.Doc(path).Set(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := editFields(client, path, []string{"age=31", "active=true"}); err != nil {
		t.Fatalf("editFields: %v", err)
	}

	snap, err := client.Doc(path).Get(ctx)
	if err != nil {
		t.Fatalf("get after edit: %v", err)
	}
	data := snap.Data()

	// Updated fields. Numbers go through parseValue as float64 and read back as
	// float64 (Firestore preserves the stored double type).
	if got := data["age"]; got != float64(31) {
		t.Errorf("age = %v (%T), want float64(31)", got, got)
	}
	if got := data["active"]; got != true {
		t.Errorf("active = %v, want true", got)
	}
	// Untouched fields survive the partial merge.
	if got := data["name"]; got != "Alice" {
		t.Errorf("name = %v, want Alice (untouched)", got)
	}
	if got := data["email"]; got != "alice@example.com" {
		t.Errorf("email = %v, want alice@example.com (untouched)", got)
	}
}

func TestEditFields_DottedPath(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	path := "edit_users/bob"
	seed := map[string]any{
		"address": map[string]any{
			"city": "LA",
			"zip":  "90001",
		},
	}
	if _, err := client.Doc(path).Set(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := editFields(client, path, []string{"address.city=NYC"}); err != nil {
		t.Fatalf("editFields: %v", err)
	}

	snap, err := client.Doc(path).Get(ctx)
	if err != nil {
		t.Fatalf("get after edit: %v", err)
	}
	addr, ok := snap.Data()["address"].(map[string]any)
	if !ok {
		t.Fatalf("address = %#v, want map", snap.Data()["address"])
	}
	if got := addr["city"]; got != "NYC" {
		t.Errorf("address.city = %v, want NYC", got)
	}
	// Sibling nested field untouched.
	if got := addr["zip"]; got != "90001" {
		t.Errorf("address.zip = %v, want 90001 (untouched)", got)
	}
}

func TestEditFields_Array(t *testing.T) {
	client := emulatorClient(t)
	ctx := context.Background()

	path := "edit_users/carol"
	if _, err := client.Doc(path).Set(ctx, map[string]any{"name": "Carol"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := editFields(client, path, []string{`tags=["a","b"]`}); err != nil {
		t.Fatalf("editFields: %v", err)
	}

	snap, err := client.Doc(path).Get(ctx)
	if err != nil {
		t.Fatalf("get after edit: %v", err)
	}
	if got := snap.Data()["tags"]; !reflect.DeepEqual(got, []any{"a", "b"}) {
		t.Errorf("tags = %#v, want [a b]", got)
	}
}

func TestEditFields_NotFound(t *testing.T) {
	client := emulatorClient(t)

	err := editFields(client, "edit_users/missing", []string{"x=1"})
	if err == nil {
		t.Fatal("expected error for missing document, got nil")
	}
	if err.Error() != "document not found" {
		t.Errorf("error = %q, want %q", err.Error(), "document not found")
	}
}
