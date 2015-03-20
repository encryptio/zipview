package main

import (
	"archive/zip"
	"io/ioutil"
	"log"
	"net/http"
	"sort"

	"honnef.co/go/js/dom"
)

func addFile(f *zip.File) {
	rc, err := f.Open()
	if err != nil {
		log.Printf("Couldn't open file in zip: %v", err)
		return
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		log.Printf("Couldn't read file in zip: %v", err)
		return
	}

	dataURL := toDataURL(http.DetectContentType(data), data)
	doc := dom.GetWindow().Document()
	img := doc.CreateElement("img")
	img.SetAttribute("src", dataURL)
	body := doc.QuerySelector("body")
	body.AppendChild(img)
}

func main() {
	url := "ookami.zip"

	r, err := OpenHTTPReader(url)
	if err != nil {
		log.Printf("Couldn't open HTTP reader: %v", err)
	}

	cr := &CacheReader{Inner: r, Size: r.Size}

	zr, err := zip.NewReader(cr, cr.Size)
	if err != nil {
		log.Printf("Couldn't open zip file: %v", err)
		return
	}

	fs := make([]*zip.File, len(zr.File))
	copy(fs, zr.File)
	sort.Sort(sortableZIPList(fs))

	go func() {
		for _, f := range fs {
			log.Printf("adding file %v", f.Name)
			addFile(f)
		}
	}()
}
