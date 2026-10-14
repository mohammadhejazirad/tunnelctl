package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// DecodeStrict rejects duplicate structural fields, trailing values, excessive
// nesting and unknown schema fields before populating the caller's model.
func DecodeStrict(r io.Reader, limit int, out any) error {
	b, e := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if e != nil {
		return e
	}
	if len(b) > limit {
		return errors.New("JSON size limit exceeded")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	if e = consumeJSON(d, 0); e != nil {
		return e
	}
	if _, e = d.Token(); e != io.EOF {
		return errors.New("trailing JSON data")
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(out)
}
func consumeJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("JSON nesting limit exceeded")
	}
	tok, e := d.Token()
	if e != nil {
		return e
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			tok, e := d.Token()
			if e != nil {
				return e
			}
			key, ok := tok.(string)
			if !ok || seen[key] {
				return errors.New("duplicate or invalid JSON field")
			}
			seen[key] = true
			if e = consumeJSON(d, depth+1); e != nil {
				return e
			}
		}
		tok, e = d.Token()
		if e != nil || tok != json.Delim('}') {
			return errors.New("invalid JSON object")
		}
	case '[':
		for d.More() {
			if e = consumeJSON(d, depth+1); e != nil {
				return e
			}
		}
		tok, e = d.Token()
		if e != nil || tok != json.Delim(']') {
			return errors.New("invalid JSON array")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}
