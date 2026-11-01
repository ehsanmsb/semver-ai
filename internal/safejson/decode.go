package safejson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Decode rejects duplicate keys, unknown fields, and trailing JSON values.
func Decode(data []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := value(d); err != nil {
		return err
	}
	if _, e := d.Token(); e != io.EOF {
		return fmt.Errorf("trailing JSON data")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func value(d *json.Decoder) error {
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
			k, e := d.Token()
			if e != nil {
				return e
			}
			s, ok := k.(string)
			if !ok || seen[s] {
				return fmt.Errorf("invalid or duplicate JSON key")
			}
			seen[s] = true
			if e = value(d); e != nil {
				return e
			}
		}
	case '[':
		for d.More() {
			if e = value(d); e != nil {
				return e
			}
		}
	default:
		return fmt.Errorf("invalid JSON delimiter")
	}
	_, e = d.Token()
	return e
}
