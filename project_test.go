package main

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestPairPackagesWithXTestsExternalFirst(t *testing.T) {
	pairs := PairPackagesWithXTests([]*packages.Package{
		{
			Name:    "widget_test",
			PkgPath: "example.test/widget_test",
		},
		{
			Name:    "widget",
			PkgPath: "example.test/widget",
		},
	})

	if len(pairs) != 1 {
		t.Fatalf("expected one paired package, got %d", len(pairs))
	}
	if _, ok := pairs["example.test/widget_test"]; ok {
		t.Fatal("external test package was stored under its unstripped package path")
	}

	pair := pairs["example.test/widget"]
	if pair == nil {
		t.Fatal("missing pair under stripped package path")
	}
	if pair.pkg == nil {
		t.Fatal("in-package package was not recorded")
	}
	if pair.test == nil {
		t.Fatal("external test package was not recorded")
	}
}
