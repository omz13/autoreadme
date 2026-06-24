package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModInfoToolchain(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, `module example.test

go 1.21.0

toolchain go1.22.0
`)

	mod, err := ModInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Toolchain != "go1.22.0" {
		t.Fatalf("Toolchain: got %q want go1.22.0", mod.Toolchain)
	}
}

func TestModInfoNoToolchain(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, `module example.test

go 1.21.0
`)

	mod, err := ModInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Toolchain != "" {
		t.Fatalf("Toolchain: got %q want empty", mod.Toolchain)
	}
}

func writeGoMod(t *testing.T, dir, contents string) {
	t.Helper()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte(contents), 0o666); err != nil {
		t.Fatal(err)
	}
}
