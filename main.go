// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// godoc2md
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"go/doc"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"text/template"

	"code.google.com/p/go.tools/godoc"
	"code.google.com/p/go.tools/godoc/vfs"
)

var (
	verbose = flag.Bool("v", false, "verbose mode")

	// file system roots
	// TODO(gri) consider the invariant that goroot always end in '/'
	goroot = flag.String("goroot", runtime.GOROOT(), "Go root directory")

	// layout control
	tabWidth       = flag.Int("tabwidth", 4, "tab width")
	showTimestamps = flag.Bool("timestamps", false, "show timestamps with directory listings")
	templateDir    = flag.String("templates", "", "directory containing alternate template files")
	showPlayground = flag.Bool("play", false, "enable playground in web interface")
	showExamples   = flag.Bool("ex", false, "show examples in command line mode")
	declLinks      = flag.Bool("links", true, "link identifiers to their declarations")
)

func usage() {
	fmt.Fprintf(os.Stderr,
		"usage: godoc2md package [name ...]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	pres *godoc.Presentation
	fs   = vfs.NameSpace{}

	funcs = map[string]interface{}{
		"comment_md": comment_mdFunc,
	}
)

const punchCardWidth = 80

func comment_mdFunc(comment string) string {
	var buf bytes.Buffer
	doc.ToText(&buf, comment, "", "\t", punchCardWidth)
	return buf.String()
}

func readTemplate(name, data string) *template.Template {
	// be explicit with errors (for app engine use)
	t, err := template.New(name).Funcs(pres.FuncMap()).Funcs(funcs).Parse(string(data))
	if err != nil {
		log.Fatal("readTemplate: ", err)
	}
	return t
}

func readTemplates(p *godoc.Presentation, html bool) {
	p.PackageText = readTemplate("package.txt", pkgTemplate)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	// use file system of underlying OS
	fs.Bind("/", vfs.OS(*goroot), "/", vfs.BindReplace)

	// Bind $GOPATH trees into Go root.
	for _, p := range filepath.SplitList(build.Default.GOPATH) {
		fs.Bind("/src/pkg", vfs.OS(p), "/src", vfs.BindAfter)
	}

	corpus := godoc.NewCorpus(fs)
	corpus.Verbose = *verbose

	pres = godoc.NewPresentation(corpus)
	pres.TabWidth = *tabWidth
	pres.ShowTimestamps = *showTimestamps
	pres.ShowPlayground = *showPlayground
	pres.ShowExamples = *showExamples
	pres.DeclLinks = *declLinks
	pres.SrcMode = false
	pres.HTMLMode = false

	readTemplates(pres, false)

	if err := godoc.CommandLine(os.Stdout, fs, pres, flag.Args()); err != nil {
		log.Print(err)
	}
}

var pkgTemplate = `{{with .PAst}}{{node $ .}}{{end}}{{/*

---------------------------------------

*/}}{{with .PDoc}}{{if $.IsMain}}# {{.ImportPath}}

{{comment_text .Doc "" "\t"}}
{{else}}# package {{.Name}}

    import "{{.ImportPath}}"

{{comment_text .Doc "" "\t"}}
{{example_text $ "" "\t"}}{{/*

---------------------------------------

*/}}{{with .Consts}}
## Constants

{{range .}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{end}}{{end}}{{/*

---------------------------------------

*/}}{{with .Vars}}
## Variables

{{range .}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{end}}{{end}}{{/*

---------------------------------------

*/}}{{with .Funcs}}
## Functions

{{range .}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{example_text $ .Name "\t"}}{{end}}{{end}}{{/*

---------------------------------------

*/}}{{with .Types}}
## Types

{{range .}}{{$tname := .Name}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{range .Consts}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{end}}{{range .Vars}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{end}}{{example_text $ .Name "    "}}
{{range .Funcs}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{example_text $ .Name "\t"}}
{{end}}{{range .Methods}}{{node $ .Decl}}
{{comment_text .Doc "" "\t"}}
{{$name := printf "%s_%s" $tname .Name}}{{example_text $ $name "    "}}{{end}}
{{end}}{{end}}{{end}}{{/*

---------------------------------------

*/}}{{with $.Notes}}
## Notes
{{range $marker, $content := .}}
{{$marker}}S

{{range $content}}{{comment_text .Body "   " "\t"}}
{{end}}{{end}}{{end}}{{end}}{{/*

---------------------------------------

*/}}`
