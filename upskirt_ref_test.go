package main

import (
	"fmt"
	"markup"
	"path/filepath"
	"io/ioutil"
	"strings"
)

const (
	testFilesDir = "testfiles"
)

var totalTested int
var failed int

func pprint(s string) {
	s = strings.Replace(s, "\n", `\n`, -1)
	s = strings.Replace(s, "\r", `\r`, -1)
	s = strings.Replace(s, "\t", `\t`, -1)
	b := []byte(s)
	if len(b) > 360 {
		b = b[:360]
	}
	s = string(b)
	fmt.Print(s)
	fmt.Print("\n")
	//fmt.Printf("%s\n", s)
}

func testStr(s string) {
	html := markup.MarkdownToHtml([]byte(s), 0)
	pprint(s)
	pprint(string(html))
}

func clean(s string) string {
	return strings.Replace(s, "\r\n", "\n", -1)
}

func testCrashFile(basename string) {
	fn := filepath.Join(testFilesDir, basename+".text")
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return
	}
	//fmt.Printf("Testing: %s\n", fn)
	html := clean(string(markup.MarkdownToHtml(src, 0)))
	//fmt.Printf("got %d:\n", len(html))
	pprint(html)
}

func testFile(basename string) bool {
	fn := filepath.Join(testFilesDir, basename+".text")
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return false
	}
	//fmt.Printf("Testing: %s\n", fn)
	fn = filepath.Join(testFilesDir, basename+"_upskirt_ref.html")
	ref, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return false
	}
	totalTested++
	html := clean(string(markup.MarkdownToHtml(src, 0)))
	htmlref := clean(string(ref))
	if html != htmlref {
		fmt.Printf("Fail: '%s'\n", basename)
		failed++

		fmt.Printf("exp %d:\n", len(htmlref))
		pprint(htmlref)
		fmt.Printf("\n")

		fmt.Printf("got %d:\n", len(html))
		pprint(html)
		fmt.Printf("\n\n")
		return false
	}
	//fmt.Printf("Ok: '%s'\n", basename)
	return true
}

func testCrashFiles() {
	files := []string{"crash00"}
	for _, basename := range files {
		testCrashFile(basename)
	}
}

func testFiles() {
	files := []string{"Amps and angle encoding", "Auto links", "Backslash escapes", "Blockquotes with code blocks", "Code Blocks", "Code Spans", "Hard-wrapped paragraphs with list-like lines", "Horizontal rules", "Inline HTML (Advanced)", "Inline HTML (Simple)", "Inline HTML comments", "Links, inline style", "Links, reference style", "Links, shortcut references", "Literal quotes in titles", "Markdown Documentation - Basics", "Markdown Documentation - Syntax", "Nested blockquotes", "Ordered and unordered lists", "Strong and em together", "Tabs", "Tidyness"}

	failed := make([]string, 0, len(files))
	succeded := make([]string, 0, len(files))
	for _, basename := range files {
		ok := testFile(basename)
		if !ok {
			failed = append(failed, basename)
		} else {
			succeded = append(succeded, basename)
		}
	}
	for _, s := range succeded {
		fmt.Printf("ok: %s\n", s)
	}
	for _, s := range failed {
		fmt.Printf("failed: %s\n", s)
	}
	totalTested := len(failed) + len(succeded)
	fmt.Printf("Failed %d out of %d tests\n", len(failed), totalTested)
}

func testStrings() {
	strings_to_test := []string{"l: <http://f.com/>.", "a [b][].\n  [b]: /url/ \"T \"qu\" ins\"", "5 > 6", "a***foo***", "b___bar___", "* 1\n* 2", "*ca", "*\ta", "foo", "_Hello World_!"}
	for _, s := range strings_to_test {
		testStr(s)
	}
}

func main() {
	//testCrashFiles()
	testFiles()
	//markup.UnitTest()
	//testStrings()
}
