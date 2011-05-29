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

func testFile(basename string) {
	fn := filepath.Join(testFilesDir, basename+".text")
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return
	}
	fn = filepath.Join(testFilesDir, basename+"_upskirt_ref.html")
	ref, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		return
	}
	totalTested++
	html := markup.MarkdownToHtml(string(src), 0)
	htmlref := string(ref)
	if html != htmlref {
		fmt.Printf("Fail: '%s'\n", basename)
		failed++
		fmt.Printf("exp %d:\n", len(htmlref))
		pprint(htmlref)
		fmt.Println(htmlref)
		fmt.Printf("got %d:", len(html))
		pprint(html)
	} else {
		fmt.Printf("Ok: '%s'\n", basename)
	}
}

func testFiles() {
	files := []string{"Tidyness"}

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
	//testFiles()
	//markup.UnitTest()
	testStrings()
}
