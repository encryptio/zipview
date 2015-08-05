package main

import (
	"strings"

	"github.com/gopherjs/gopherjs/js"
)

func urlChannel() <-chan string {
	ch := make(chan string, 1)

	handleHashChange := func() {
		hash := js.Global.Get("location").Get("hash").String()
		hash = strings.TrimPrefix(hash, "#")
		ch <- hash
	}
	handleHashChange()

	js.Global.Call("addEventListener", "hashchange", handleHashChange, false)

	return ch
}

func main() {
	urls := urlChannel()

	viewer := NewViewer()
	for url := range urls {
		viewer.SetURL(url)
	}
}
