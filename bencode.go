package seadex

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
)

type bencodeDict map[string]any

type bencodeParser struct {
	data []byte
	pos  int
}

func parseBencode(data []byte) (any, error) {
	p := &bencodeParser{data: data}
	v, err := p.parse()
	if err != nil {
		return nil, err
	}
	if p.pos != len(data) {
		return nil, fmt.Errorf("trailing data at byte %d", p.pos)
	}
	return v, nil
}

func (p *bencodeParser) parse() (any, error) {
	if p.pos >= len(p.data) {
		return nil, io.ErrUnexpectedEOF
	}
	switch c := p.data[p.pos]; {
	case c == 'i':
		return p.parseInt()
	case c == 'l':
		return p.parseList()
	case c == 'd':
		return p.parseDict()
	case c >= '0' && c <= '9':
		return p.parseString()
	default:
		return nil, fmt.Errorf("invalid bencode token %q at byte %d", c, p.pos)
	}
}

func (p *bencodeParser) parseInt() (int64, error) {
	p.pos++
	end := bytes.IndexByte(p.data[p.pos:], 'e')
	if end < 0 {
		return 0, io.ErrUnexpectedEOF
	}
	raw := string(p.data[p.pos : p.pos+end])
	p.pos += end + 1
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", raw, err)
	}
	return n, nil
}

func (p *bencodeParser) parseString() (string, error) {
	colon := bytes.IndexByte(p.data[p.pos:], ':')
	if colon < 0 {
		return "", io.ErrUnexpectedEOF
	}
	rawLen := string(p.data[p.pos : p.pos+colon])
	n, err := strconv.Atoi(rawLen)
	if err != nil || n < 0 {
		return "", fmt.Errorf("invalid string length %q", rawLen)
	}
	p.pos += colon + 1
	if p.pos+n > len(p.data) {
		return "", io.ErrUnexpectedEOF
	}
	s := string(p.data[p.pos : p.pos+n])
	p.pos += n
	return s, nil
}

func (p *bencodeParser) parseList() ([]any, error) {
	p.pos++
	var out []any
	for {
		if p.pos >= len(p.data) {
			return nil, io.ErrUnexpectedEOF
		}
		if p.data[p.pos] == 'e' {
			p.pos++
			return out, nil
		}
		v, err := p.parse()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

func (p *bencodeParser) parseDict() (bencodeDict, error) {
	p.pos++
	out := make(bencodeDict)
	for {
		if p.pos >= len(p.data) {
			return nil, io.ErrUnexpectedEOF
		}
		if p.data[p.pos] == 'e' {
			p.pos++
			return out, nil
		}
		key, err := p.parseString()
		if err != nil {
			return nil, fmt.Errorf("parsing dict key: %w", err)
		}
		v, err := p.parse()
		if err != nil {
			return nil, fmt.Errorf("parsing dict value for %q: %w", key, err)
		}
		out[key] = v
	}
}

func encodeBencode(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeBencode(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeBencode(w *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case string:
		fmt.Fprintf(w, "%d:%s", len(x), x)
	case int:
		fmt.Fprintf(w, "i%de", x)
	case int64:
		fmt.Fprintf(w, "i%de", x)
	case []any:
		w.WriteByte('l')
		for _, item := range x {
			if err := writeBencode(w, item); err != nil {
				return err
			}
		}
		w.WriteByte('e')
	case bencodeDict:
		w.WriteByte('d')
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := writeBencode(w, k); err != nil {
				return err
			}
			if err := writeBencode(w, x[k]); err != nil {
				return err
			}
		}
		w.WriteByte('e')
	default:
		return fmt.Errorf("unsupported bencode value %T", v)
	}
	return nil
}
