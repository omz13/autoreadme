package main

import (
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

type Module struct {
	Path          string
	Version       string
	Deprecated    string
	GoVersion     string
	Toolchain     string
	Documentation *Doc
}

func ModInfo(projectRoot string) (*Module, error) {
	filePath := filepath.Join(projectRoot, "go.mod")
	bs, err := readFile(filePath)
	if err != nil {
		return nil, err
	}
	f, err := modfile.ParseLax(filePath, bs, nil)
	if err != nil {
		return nil, err
	}

	mod := f.Module.Mod
	Path := mod.Path
	Version := mod.Version

	Deprecated := f.Module.Deprecated

	GoVersion := ""
	if f.Go != nil {
		GoVersion = f.Go.Version
	}

	Toolchain := moduleToolchain(f, bs)

	comments := f.Module.Syntax.Comments.Before
	text := stripDeprecation(flattenModComments(comments))

	return &Module{
		Path:          Path,
		Version:       Version,
		Deprecated:    Deprecated,
		GoVersion:     GoVersion,
		Toolchain:     Toolchain,
		Documentation: NewDoc(text),
	}, nil
}

func flattenModComments(lines []modfile.Comment) string {
	var acc []string
	for _, line := range lines {
		text := line.Token[2:]
		acc = append(acc, text)
	}
	return strings.Join(acc, "\n")
}

// adapted from deprecatedRE in modfile's rule.go
var deprecatedRE = regexp.MustCompile(`(?ms)((^[ \t]*|\n\n)Deprecated: *(.*?)($|\n\n))`)

func stripDeprecation(text string) string {
	return deprecatedRE.ReplaceAllString(text, "\n")
}

// moduleToolchain returns the toolchain directive from go.mod. ParseLax omits
// it even though the statement is known (golang/go#67132), so fall back to a
// line scan of the file we already read.
func moduleToolchain(f *modfile.File, src []byte) string {
	if f.Toolchain != nil {
		return f.Toolchain.Name
	}
	return modfileToolchainFromSource(src)
}

var toolchainLineRE = regexp.MustCompile(`(?m)^[ \t]*toolchain[ \t]+(\S+)`)

func modfileToolchainFromSource(src []byte) string {
	m := toolchainLineRE.FindSubmatch(src)
	if len(m) < 2 {
		return ""
	}
	return string(m[1])
}
