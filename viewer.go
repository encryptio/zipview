package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

type AsyncImage struct {
	doc  dom.Document
	file *zip.File

	Image dom.Element
	Err   error
	Done  chan struct{}
}

func LoadAsyncImage(doc dom.Document, file *zip.File) *AsyncImage {
	ai := &AsyncImage{
		doc:  doc,
		file: file,
		Done: make(chan struct{}),
	}
	fmt.Printf("starting AsyncImage for %v\n", file.Name)
	go ai.load()
	return ai
}

func (ai *AsyncImage) load() {
	defer close(ai.Done)
	defer fmt.Printf("finished AsyncImage for %v\n", ai.file.Name)

	rc, err := ai.file.Open()
	if err != nil {
		ai.Err = err
		return
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		ai.Err = err
		return
	}

	dataURL := toDataURL(http.DetectContentType(data), data)
	img := ai.doc.CreateElement("img")
	img.SetAttribute("src", dataURL)
	ai.Image = img
}

type windowedImages struct {
	doc        dom.Document
	files      []*zip.File
	cache      map[int]*AsyncImage
	lastLoaded int
}

func (w *windowedImages) load(index int) *AsyncImage {
	ai := w.cache[index]
	if ai == nil {
		ai = LoadAsyncImage(w.doc, w.files[index])
		w.cache[index] = ai
	}

	w.lastLoaded = index
	go func() {
		<-ai.Done
		if w.lastLoaded == index {
			if w.cache[index-1] == nil && index > 0 {
				w.cache[index-1] = LoadAsyncImage(w.doc, w.files[index-1])
			}
			if w.cache[index+1] == nil && index < len(w.files)-1 {
				w.cache[index+1] = LoadAsyncImage(w.doc, w.files[index+1])
			}
		}
	}()

	for i := range w.cache {
		if i < index-2 || i > index+2 {
			delete(w.cache, i)
		}
	}

	return ai
}

type Viewer struct {
	done chan struct{}

	doc    dom.Document
	rootEl dom.Element

	url           string
	err           error
	files         []*zip.File
	imageWindow   *windowedImages
	newAsyncImage chan *AsyncImage
	loading       bool
	image         dom.Element
	imageErr      error
	index         int
}

func NewViewer() *Viewer {
	doc := dom.GetWindow().Document()
	rootEl := doc.CreateElement("div")
	rootEl.Class().Add("viewerBase")
	doc.QuerySelector("body").AppendChild(rootEl)

	v := &Viewer{
		doc:           doc,
		rootEl:        rootEl,
		newAsyncImage: make(chan *AsyncImage),
		done:          make(chan struct{}),
	}

	js.Global.Call("addEventListener", "resize", v.Render, false)
	v.Render()

	js.Global.Call("addEventListener", "keydown", v.keydown, false)
	js.Global.Call("addEventListener", "click", v.click, false)

	go v.loadLoop()

	return v
}

func (v *Viewer) keydown(evt *js.Object) {
	if v.files == nil {
		return
	}

	switch evt.Get("keyCode").Int() {
	case 37:
		// left
		evt.Call("preventDefault")
		evt.Call("stopPropagation")

		go v.left()
	case 39:
		// right
		evt.Call("preventDefault")
		evt.Call("stopPropagation")

		go v.right()
	}
}

func (v *Viewer) click(evt *js.Object) {
	if v.files == nil {
		return
	}

	x := evt.Get("clientX").Int()
	w := js.Global.Get("innerWidth").Int()
	fmt.Printf("click, x = %v, w = %v\n", x, w)
	if x > w/2 {
		go v.right()
	} else {
		go v.left()
	}
	evt.Call("preventDefault")
	evt.Call("stopPropagation")
}

func (v *Viewer) left() {
	if v.index > 0 {
		v.index--
		v.newAsyncImage <- v.imageWindow.load(v.index)
	}
}

func (v *Viewer) right() {
	if v.index < len(v.files)-1 {
		v.index++
		v.newAsyncImage <- v.imageWindow.load(v.index)
	}
}

func (v *Viewer) loadLoop() {
	var ai *AsyncImage
	var aiDone chan struct{}
	for {
		select {
		case <-v.done:
			return
		case newAI := <-v.newAsyncImage:
			ai = newAI
			aiDone = ai.Done
			v.loading = true
			v.Render()
		case <-aiDone:
			v.imageErr = ai.Err
			v.image = ai.Image
			v.loading = false
			v.Render()
			aiDone = nil
			ai = nil
		}
	}
}

func (v *Viewer) Close() error {
	v.rootEl.ParentElement().RemoveChild(v.rootEl)
	v.rootEl = nil
	close(v.done)
	return nil
}

func (v *Viewer) SetURL(url string) {
	if v.url == url {
		return
	}

	defer v.Render()

	v.url = url
	v.files = nil
	v.imageWindow = nil
	v.image = nil
	v.imageErr = nil
	v.Render()

	r, err := OpenHTTPReader(url)
	if err != nil {
		v.err = err
		return
	}

	cr := &CacheReader{Inner: r, Size: r.Size}

	zr, err := zip.NewReader(cr, cr.Size)
	if err != nil {
		v.err = fmt.Errorf("Couldn't open zip file: %v", err)
		return
	}

	if len(zr.File) == 0 {
		v.err = fmt.Errorf("No files in zip")
		return
	}

	fs := make([]*zip.File, len(zr.File))
	copy(fs, zr.File)
	sort.Sort(sortableZIPList(fs))

	v.files = fs
	v.imageWindow = &windowedImages{
		doc:   v.doc,
		files: v.files,
		cache: make(map[int]*AsyncImage, 4),
	}
	v.index = 0
	v.newAsyncImage <- v.imageWindow.load(v.index)
}

func (v *Viewer) Render() {
	for {
		child := v.rootEl.FirstChild()
		if child == nil {
			break
		}
		v.rootEl.RemoveChild(child)
	}

	height := js.Global.Get("innerHeight").Int()
	v.rootEl.SetAttribute("style", "height: "+strconv.FormatInt(int64(height), 10)+"px")

	if v.url == "" {
		v.renderText("No URL set")
		return
	}

	if v.err != nil {
		v.renderText(v.err.Error())
		return
	}

	if v.files == nil {
		v.renderText("Opening " + v.url)
		return
	}

	context := v.doc.CreateElement("div")
	context.Class().Add("viewerContext")
	context.AppendChild(v.doc.CreateTextNode(fmt.Sprintf("[%v/%v] %v", v.index+1, len(v.files), v.files[v.index].Name)))
	if v.loading {
		context.Class().Add("loading")
		context.AppendChild(v.doc.CreateTextNode(" (loading)"))
	}
	v.rootEl.AppendChild(context)

	if v.imageErr != nil {
		v.renderText(fmt.Sprintf("Couldn't open %v in zip file: %v", v.files[v.index].Name, v.imageErr))
		return
	}

	if v.image != nil {
		v.rootEl.AppendChild(v.image)
		return
	}

	v.renderText(fmt.Sprintf("Loading %v", v.files[v.index].Name))
}

func (v *Viewer) renderText(text string) {
	message := v.doc.CreateElement("div")
	message.Class().Add("viewerMessage")
	message.AppendChild(v.doc.CreateTextNode(text))
	v.rootEl.AppendChild(message)
}
