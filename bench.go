package main

import (
	"fmt"
	"io/ioutil"
	"markup"
	"os"
	"path/filepath"
	"time"
)

const ITER = 200
const ITER2 = 10

func main() {
	// this is the largest 27k file that I have
	fn := filepath.Join("testfiles", "Markdown Documentation - Syntax.text")
	d, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Printf("Couldn't open '%s', error: %v\n", fn, err)
		os.Exit(1)
	}

	var start, end, dur, minDur int64

	minDur = 1e9
	for j := 0; j < ITER2; j++ {
		start = time.Nanoseconds()
		for i:= 0; i < ITER; i++ {
			markup.MarkdownToHtml(d, 0)
		}
		end = time.Nanoseconds()
		dur = (end - start) / 10e6
		if dur < minDur {
			minDur = dur
		}
	}
	fmt.Printf("Converting, no extensions : %v ms\n", minDur)

	minDur = 1e9
	for j := 0; j < ITER2; j++ {
		start = time.Nanoseconds()
		for i:= 0; i < ITER; i++ {
			markup.MarkdownToHtml(d, 0xff)
		}
		end = time.Nanoseconds()
		dur = (end - start) / 10e6
		if dur < minDur {
			minDur = dur
		}
	}
	fmt.Printf("Converting, all extensions: %v ms\n", minDur)

}