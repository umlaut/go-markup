package main

import (
	"fmt"
	"markup"
	"path/filepath"
	"io/ioutil"
)

const (
	testFilesDir = "testfiles"
)

var totalTested int
var failed int

func testStr(s string) {
	html := markup.MarkdownToHtml(s, 0)
	fmt.Printf("Result '%s' => '%s'\n",s, html)
}

func testFile(basename string) {
	src := filepath.Join(testFilesDir, basename+".text")
	htmlref := filepath.Join(testFilesDir, basename+"_upskirt_ref.html")
	srcdata, err := ioutil.ReadFile(src)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", src, err)
		return
	}
	refdata, err := ioutil.ReadFile(htmlref)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", htmlref, err)
		return
	}
	totalTested++
	s := string(srcdata)
	html := markup.MarkdownToHtml(s, 0)
	htmlrefstr := string(refdata)
	if html != htmlrefstr {
		fmt.Printf("Fail: '%s'\n", basename)
		failed++
		fmt.Printf("Got:\n%s\n", html)
		fmt.Printf("Expected:\n%s\n", htmlrefstr)
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
	strings_to_test := []string{ "*	a", "foo", "_Hello World_!" }
	for _, s := range strings_to_test {
		testStr(s)
	}
}

func main() {
	//testFiles()
	testStrings()
}
