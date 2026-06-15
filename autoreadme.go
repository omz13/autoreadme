package main

import (
	"bytes"
	"cmp"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
)

var PrintTemplate = flag.Bool("print-template", false, "write the built in template to stdout and exit")
var Version = flag.Bool("version", false, "output version information")
var Check = flag.Bool("check", false, "report README.md files that are out of date and exit non-zero without writing anything")
var Verbose = flag.Bool("v", false, "report every package as written, unchanged, or skipped")

func main() {
	log.SetFlags(0)
	flag.Usage = func() {
		log.Printf("Usage of %s:", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()

	if *PrintTemplate {
		fmt.Println(defaultTemplateSrc)
		return
	}
	if *Version {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("no version information in build")
			return
		}
		m := info.Main
		sum := ""
		if m.Sum != "" {
			sum = fmt.Sprintf(" (%s)", m.Sum)
		}
		fmt.Printf("%s%s, built with %s\n", cmp.Or(m.Version, "unknown version"), sum, info.GoVersion)
		return
	}

	ctx := context.Background()

	err := Main(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func Main(ctx context.Context) error {
	defaultTemplate, err := parseTemplate(defaultTemplateSrc)
	if err != nil {
		return err
	}

	repo, project, err := Roots()
	if err != nil {
		return err
	}

	// get any global config from the repo

	repoTemplateSrc, err := RepoTemplate(repo)
	if err != nil {
		return err
	}

	// if there's a repo-level template, override the default template
	defaultTemplate, err = parseTemplateOr(repoTemplateSrc, defaultTemplate)
	if err != nil {
		return err
	}

	repoData, err := RepoData(repo)
	if err != nil {
		return err
	}

	ignore, err := RepoIgnore(repo)
	if err != nil {
		return err
	}

	mod, err := ModInfo(repo)
	if err != nil {
		return err
	}

	// get all the packages in this module

	fset, packages, err := Packages(ctx, project)
	if err != nil {
		return err
	}

	// associate X and X_test packages
	pairs := PairPackagesWithXTests(packages)
	// filter out any ignored packages
	for k := range ignore {
		if _, ok := pairs[k]; ok && *Verbose {
			log.Printf("skipped (ignored): %s", k)
		}
		delete(pairs, k)
	}

	// halt on any errors, in packages we plan to inspect
	errs := CollectProjectErrors(pairs)
	if len(errs) > 0 {
		for _, err := range errs {
			log.Println(err)
		}
		return fmt.Errorf("project contains errors, cannot proceed")
	}

	// grab directory info and compute local info from what's been gathered so far
	for imp, info := range pairs {
		if err := ProcessPackageDir(fset, info, defaultTemplate); err != nil {
			return fmt.Errorf("preparing %s: %w", imp, err)
		}
	}

	type Repository struct {
		Data any
	}
	repository := &Repository{
		Data: repoData,
	}
	type Context struct {
		Repository  *Repository
		Module      *Module
		Package     *Package
		ProjectRoot bool
	}

	// Execute the templates and, if they result in a change, queue them up for output
	type file struct {
		path     string
		contents []byte
	}
	var files []file
	var buf bytes.Buffer
	for imp, info := range pairs {
		buf.Reset()

		context := &Context{
			Repository:  repository,
			Module:      mod,
			Package:     PackageFromInfo(fset, info),
			ProjectRoot: repo == info.dir,
		}

		err := info.template.Execute(&buf, context)
		if err != nil {
			return fmt.Errorf("template for %s failed: %w", imp, err)
		}

		contents := buf.Bytes()
		path := filepath.Join(info.dir, "README.md")
		// we only queue the write if there's been a change to the contents
		if !bytes.Equal(contents, info.oldReadme) {
			files = append(files, file{
				path:     path,
				contents: bytes.Clone(contents),
			})
		} else if *Verbose {
			log.Printf("unchanged: %s", path)
		}
	}

	// pairs is a map, so the queue order is nondeterministic; sort it so -check
	// and -v output is stable across runs (CI logs, diffs).
	slices.SortFunc(files, func(a, b file) int {
		return strings.Compare(a.path, b.path)
	})

	// -check is a dry run: report what's stale, write nothing, exit non-zero if
	// anything is out of date. The CI guard against a forgotten go:generate.
	if *Check {
		for _, file := range files {
			log.Printf("out of date: %s", file.path)
		}
		if len(files) > 0 {
			return fmt.Errorf("%d README.md file(s) out of date; run go generate", len(files))
		}
		if *Verbose {
			log.Printf("all README.md files up to date")
		}
		return nil
	}

	for _, file := range files {
		if *Verbose {
			log.Printf("writing: %s", file.path)
		}
		if err := os.WriteFile(file.path, file.contents, 0666); err != nil {
			return err
		}
	}

	return nil
}
