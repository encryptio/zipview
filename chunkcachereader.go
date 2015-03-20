package main

import (
	"io"
)

const (
	// max out at 16MiB of cache (per CacheReader)

	chunkSize  = 262144
	chunkCount = 4 * 16
)

type CacheReader struct {
	Inner io.ReaderAt
	Size  int64

	parts        [][]byte
	partsUsed    []int
	partsUsedMap map[int]struct{}
}

func (cr *CacheReader) ReadAt(p []byte, off int64) (n int, err error) {
	if cr.parts == nil {
		cr.parts = make([][]byte, (cr.Size+chunkSize-1)/chunkSize)
		cr.partsUsed = make([]int, 0, chunkCount+1)
		cr.partsUsedMap = make(map[int]struct{}, chunkCount+1)
	}

	partIdx := int(off / chunkSize)
	offset := off - int64(partIdx)*chunkSize
	for len(p) > 0 {
		if partIdx >= len(cr.parts) {
			return n, io.EOF
		}

		if cr.parts[partIdx] == nil {
			wantSize := chunkSize
			if partIdx == len(cr.parts)-1 {
				wantSize = int(cr.Size - int64(partIdx)*chunkSize)
			}

			data := make([]byte, wantSize)
			rn, rerr := cr.Inner.ReadAt(data, int64(partIdx)*chunkSize)
			if rerr == io.EOF && rn == wantSize {
				rerr = nil
			}
			if err == io.EOF {
				rerr = io.ErrUnexpectedEOF
			}
			if rerr != nil {
				return n, err
			}

			cr.parts[partIdx] = data
		}

		rd := copy(p, cr.parts[partIdx][offset:])
		p = p[rd:]
		n += rd

		if _, ok := cr.partsUsedMap[partIdx]; !ok {
			cr.partsUsed = append(cr.partsUsed, partIdx)
			cr.partsUsedMap[partIdx] = struct{}{}

			if len(cr.partsUsed) > chunkCount {
				removeIdx := cr.partsUsed[0]
				copy(cr.partsUsed, cr.partsUsed[1:])
				cr.partsUsed = cr.partsUsed[:len(cr.partsUsed)-1]
				delete(cr.partsUsedMap, removeIdx)
				cr.parts[removeIdx] = nil
			}
		}

		partIdx++
		offset = 0
	}

	return n, nil
}
