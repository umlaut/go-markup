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
	fmt.Print(s)
	fmt.Print("\n")
	//fmt.Printf("%s\n", s)
}

func testStr(s string) {
	html := markup.MarkdownToHtml(s, 0)
	pprint(s)
	pprint(html)
}

func clean(s string) string {
	return strings.Replace(s, "\r\n", "\n", -1)
}

func testFile(basename string) {
	fn := filepath.Join(testFilesDir, basename+".text")
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return
	}
	fmt.Printf("Testing: %s\n", fn)
	fn = filepath.Join(testFilesDir, basename+"_upskirt_ref.html")
	ref, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return
	}
	totalTested++
	html := clean(markup.MarkdownToHtml(string(src), 0))
	htmlref := clean(string(ref))
	if html != htmlref {
		fmt.Printf("Fail: '%s'\n", basename)
		failed++

		fmt.Printf("exp %d:\n", len(htmlref))
		pprint(htmlref)
		//fmt.Println(htmlref)

		fmt.Printf("got %d:\n", len(html))
		pprint(html)
		//fmt.Println(html)
	} else {
		fmt.Printf("Ok: '%s'\n", basename)
	}
}

func testFiles() {
	files := []string{"Amps and angle encoding", "Auto links", "Backslash escapes", "Blockquotes with code blocks", "Code Blocks", "Code Spans", "Hard-wrapped paragraphs with list-like lines", "Horizontal rules", "Inline HTML (Advanced)", "Inline HTML (Simple)", "Inline HTML comments", "Links, inline style", "Links, reference style", "Links, shortcut references", "Literal quotes in titles", "Markdown Documentation - Basics", "Markdown Documentation - Syntax", "Nested blockquotes", "Ordered and unordered lists", "Strong and em together", "Tabs", "Tidyness"}

	totalTested = 0
	failed = 0
	for _, basename := range files {
		testFile(basename)
	}
	fmt.Printf("Failed %d out of %d tests\n", failed, totalTested)
}

func testStrings() {
	strings_to_test := []string{"* 1\n* 2", "*ca", "*\ta", "foo", "_Hello World_!"}
	for _, s := range strings_to_test {
		testStr(s)
	}
}

func main() {
	testFiles()
	//markup.UnitTest()
	//testStrings()
}
