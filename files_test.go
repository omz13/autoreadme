package main

import "testing"

func TestReadFileReturnsUnexpectedErrors(t *testing.T) {
	_, err := readFile(t.TempDir())
	if err == nil {
		t.Fatal("expected error when reading a directory")
	}
}
