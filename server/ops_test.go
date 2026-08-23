package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReloadOperators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ops.txt")
	if err := os.WriteFile(path, []byte("Alice\n"), 0600); err != nil {
		t.Fatal(err)
	}
	srv := &Server{ops: loadOperators(path)}
	if !srv.IsOp("alice") {
		t.Fatal("initial operator missing")
	}
	if err := os.WriteFile(path, []byte("Bob\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := srv.ReloadOperators(); err != nil {
		t.Fatalf("reload operators: %v", err)
	}
	if srv.IsOp("alice") || !srv.IsOp("BOB") {
		t.Fatal("operator reload did not replace names")
	}
}
