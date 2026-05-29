package seadex

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SeaDexTorrent struct {
	file string
	root bencodeDict
	info bencodeDict
}

func NewSeaDexTorrent(filePath string) (*SeaDexTorrent, error) {
	resolved, err := realTorrentPath(filePath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	raw, err := parseBencode(data)
	if err != nil {
		return nil, fmt.Errorf("parsing torrent: %w", err)
	}
	root, ok := raw.(bencodeDict)
	if !ok {
		return nil, fmt.Errorf("torrent root must be a dictionary")
	}
	info, ok := root["info"].(bencodeDict)
	if !ok {
		return nil, fmt.Errorf("torrent info must be a dictionary")
	}
	return &SeaDexTorrent{file: resolved, root: root, info: info}, nil
}

func realTorrentPath(filePath string) (string, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if os.IsNotExist(err) {
		return abs, nil
	}
	return "", err
}

func (t *SeaDexTorrent) File() string {
	return t.file
}

func (t *SeaDexTorrent) IsPrivate() bool {
	private, ok := t.info["private"].(int64)
	return ok && private == 1
}

func (t *SeaDexTorrent) FileList() ([]File, error) {
	if rawFiles, ok := t.info["files"].([]any); ok {
		files := make([]File, 0, len(rawFiles))
		for _, raw := range rawFiles {
			d, ok := raw.(bencodeDict)
			if !ok {
				return nil, fmt.Errorf("torrent file entry must be a dictionary")
			}
			size, err := bencodeInt(d, "length")
			if err != nil {
				return nil, err
			}
			pathParts, err := bencodeStringList(d, "path")
			if err != nil {
				return nil, err
			}
			files = append(files, File{Name: strings.Join(pathParts, "/"), Size: size})
		}
		return files, nil
	}

	name, err := bencodeString(t.info, "name")
	if err != nil {
		return nil, err
	}
	size, err := bencodeInt(t.info, "length")
	if err != nil {
		return nil, err
	}
	return []File{{Name: filepath.ToSlash(name), Size: size}}, nil
}

func (t *SeaDexTorrent) Sanitize(destination string, overwrite bool) (string, error) {
	if !t.IsPrivate() {
		return t.File(), nil
	}

	delete(t.root, "announce")
	delete(t.root, "announce-list")
	delete(t.root, "url-list")
	delete(t.root, "httpseeds")
	delete(t.root, "comment")
	delete(t.root, "creation date")
	delete(t.root, "created by")
	delete(t.info, "private")
	delete(t.info, "source")

	entropy := make([]byte, 16)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}
	t.info["seadex-infohash-randomizer"] = hex.EncodeToString(entropy)

	out := destination
	if out == "" {
		if !overwrite {
			return "", &os.PathError{Op: "sanitize", Path: t.file, Err: os.ErrExist}
		}
		out = t.file
	} else {
		resolved, err := realTorrentPath(out)
		if err != nil {
			return "", err
		}
		out = resolved
		if !overwrite {
			if _, err := os.Stat(out); err == nil {
				return "", &os.PathError{Op: "sanitize", Path: out, Err: os.ErrExist}
			} else if !os.IsNotExist(err) {
				return "", err
			}
		}
	}

	encoded, err := encodeBencode(t.root)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(out, encoded, 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func bencodeString(d bencodeDict, key string) (string, error) {
	v, ok := d[key].(string)
	if !ok {
		return "", fmt.Errorf("torrent field %q must be a string", key)
	}
	return v, nil
}

func bencodeInt(d bencodeDict, key string) (int64, error) {
	v, ok := d[key].(int64)
	if !ok {
		return 0, fmt.Errorf("torrent field %q must be an integer", key)
	}
	return v, nil
}

func bencodeStringList(d bencodeDict, key string) ([]string, error) {
	raw, ok := d[key].([]any)
	if !ok {
		return nil, fmt.Errorf("torrent field %q must be a list", key)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("torrent field %q must contain strings", key)
		}
		out = append(out, s)
	}
	return out, nil
}
