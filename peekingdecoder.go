package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
)

type peekingDecoder struct {
	*json.Decoder
	rd *bufio.Reader
}

func newPeekingDecoder(r io.Reader) *peekingDecoder {
	br := bufio.NewReader(r)
	dec := json.NewDecoder(br)
	dec.UseNumber()
	return &peekingDecoder{
		dec,
		br,
	}
}

func (d *peekingDecoder) NextByte() (byte, error) {
	bs := make([]byte, 1)
	br := d.Decoder.Buffered()

	// Need to loop here, because we will get back spaces and stuff between
	// the objects when reading.
	for {
		n, err := br.Read(bs)
		if err != nil || n != 1 {
			break
		}
		if len(bytes.TrimSpace(bs)) == 0 {
			continue
		}
		return bs[0], nil
	}

	// OK, nothing in the json.Decoder buffer, lets peek in the buffered
	// reader instead.
	for {
		if bs, err := d.rd.Peek(1); err != nil {
			return 0, err
		} else {
			if len(bytes.TrimSpace(bs)) == 0 {
				// It was a space
				d.rd.Discard(1)
				continue
			}
			return bs[0], nil
		}
	}
}
