package seadex

import (
	"crypto/sha1"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTestTorrent(t *testing.T, path string, private bool) {
	t.Helper()
	info := bencodeDict{
		"files": []any{
			bencodeDict{"length": int64(100), "path": []any{"Season", "Ep.mkv"}},
			bencodeDict{"length": int64(200), "path": []any{"Season", "OVA.mkv"}},
		},
		"name": "Release",
	}
	if private {
		info["private"] = int64(1)
		info["source"] = "Private"
	}
	data, err := encodeBencode(bencodeDict{
		"announce":      "https://tracker.example/announce",
		"announce-list": []any{[]any{"https://tracker.example/announce"}},
		"created by":    "unittest",
		"comment":       "test",
		"creation date": int64(12345),
		"httpseeds":     []any{"https://example/httpseed"},
		"info":          info,
		"url-list":      []any{"https://example/webseed"},
	})
	if err != nil {
		t.Fatalf("encoding torrent: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing torrent: %v", err)
	}
}

func torrentInfoHash(t *testing.T, path string) [sha1.Size]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading torrent: %v", err)
	}
	raw, err := parseBencode(data)
	if err != nil {
		t.Fatalf("parsing torrent: %v", err)
	}
	root, ok := raw.(bencodeDict)
	if !ok {
		t.Fatal("torrent root is not a dictionary")
	}
	info, ok := root["info"].(bencodeDict)
	if !ok {
		t.Fatal("torrent info is not a dictionary")
	}
	encoded, err := encodeBencode(info)
	if err != nil {
		t.Fatalf("encoding info: %v", err)
	}
	return sha1.Sum(encoded)
}

func TestSeaDexTorrentFileList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.torrent")
	writeTestTorrent(t, path, true)

	torrent, err := NewSeaDexTorrent(path)
	if err != nil {
		t.Fatalf("NewSeaDexTorrent: %v", err)
	}
	files, err := torrent.FileList()
	if err != nil {
		t.Fatalf("FileList: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Name != "Season/Ep.mkv" || files[0].Size != 100 {
		t.Errorf("files[0] = %#v", files[0])
	}
	if files[1].Name != "Season/OVA.mkv" || files[1].Size != 200 {
		t.Errorf("files[1] = %#v", files[1])
	}
}

func TestSeaDexTorrentSanitizePrivate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "private.torrent")
	dst := filepath.Join(dir, "public.torrent")
	writeTestTorrent(t, src, true)
	originalHash := torrentInfoHash(t, src)

	torrent, err := NewSeaDexTorrent(src)
	if err != nil {
		t.Fatalf("NewSeaDexTorrent: %v", err)
	}
	if !torrent.IsPrivate() {
		t.Fatal("IsPrivate() = false, want true")
	}
	out, err := torrent.Sanitize(dst, false)
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	if out != dst {
		t.Errorf("Sanitize path = %q, want %q", out, dst)
	}

	sanitized, err := NewSeaDexTorrent(dst)
	if err != nil {
		t.Fatalf("reading sanitized torrent: %v", err)
	}
	if sanitized.IsPrivate() {
		t.Fatal("sanitized torrent should not be private")
	}
	if originalHash == torrentInfoHash(t, dst) {
		t.Fatal("expected sanitized info hash to change")
	}
}

func TestSeaDexTorrentSanitizePublicPassthrough(t *testing.T) {
	src := filepath.Join(t.TempDir(), "public.torrent")
	writeTestTorrent(t, src, false)

	torrent, err := NewSeaDexTorrent(src)
	if err != nil {
		t.Fatalf("NewSeaDexTorrent: %v", err)
	}
	out, err := torrent.Sanitize(filepath.Join(t.TempDir(), "ignored.torrent"), false)
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	if out != src {
		t.Errorf("public sanitize returned %q, want %q", out, src)
	}
}

func TestSeaDexTorrentSanitizeOverwriteRequired(t *testing.T) {
	src := filepath.Join(t.TempDir(), "private.torrent")
	writeTestTorrent(t, src, true)

	torrent, err := NewSeaDexTorrent(src)
	if err != nil {
		t.Fatalf("NewSeaDexTorrent: %v", err)
	}
	_, err = torrent.Sanitize("", false)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("Sanitize error = %v, want os.ErrExist", err)
	}
}
