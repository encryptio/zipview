package main

import (
	"fmt"
	"log"
	"net/http"
)

type HTTPReader struct {
	URL  string
	Size int64
}

func OpenHTTPReader(url string) (*HTTPReader, error) {
	log.Printf("openhttpreader")
	resp, err := http.Head(url)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("couldn't HEAD %v, got response code %v",
			url, resp.StatusCode)
	}

	if resp.ContentLength < 0 {
		return nil, fmt.Errorf("%v has unknown content-length", url)
	}
	if resp.ContentLength == 0 {
		return nil, fmt.Errorf("%v has zero content-length", url)
	}
	log.Printf("head complete, content length is %v", resp.ContentLength)

	return &HTTPReader{url, resp.ContentLength}, nil
}

func (h *HTTPReader) ReadAt(p []byte, off int64) (n int, err error) {
	req, err := http.NewRequest("GET", h.URL, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", off, off+int64(len(p))))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	for len(p) > 0 {
		rn, rerr := resp.Body.Read(p)
		n += rn
		p = p[rn:]
		if rerr != nil {
			resp.Body.Close()
			return n, rerr
		}
	}

	err = resp.Body.Close()
	return n, err
}
