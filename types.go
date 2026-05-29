package seadex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (f File) String() string {
	return f.Name
}

func (f File) ToDict() map[string]any {
	return map[string]any{
		"name": f.Name,
		"size": f.Size,
	}
}

func (t TorrentRecord) ToDict() map[string]any {
	files := make([]map[string]any, len(t.Files))
	for i, f := range t.Files {
		files[i] = f.ToDict()
	}
	tags := make([]string, len(t.Tags))
	for i, tag := range t.Tags {
		tags[i] = string(tag)
	}
	var infohash any
	if t.Infohash != nil {
		infohash = *t.Infohash
	}
	var groupedURL any
	if t.GroupedURL != nil {
		groupedURL = *t.GroupedURL
	}
	return map[string]any{
		"collection_id":   t.CollectionID,
		"collection_name": t.CollectionName,
		"created_at":      t.CreatedAt,
		"is_dual_audio":   t.IsDualAudio,
		"files":           files,
		"id":              t.ID,
		"infohash":        infohash,
		"is_best":         t.IsBest,
		"release_group":   t.ReleaseGroup,
		"tags":            tags,
		"tracker":         string(t.Tracker),
		"updated_at":      t.UpdatedAt,
		"url":             t.URL,
		"grouped_url":     groupedURL,
		"size":            t.Size,
	}
}

func (t TorrentRecord) ToJSON() (string, error) {
	b, err := marshalIndent(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func TorrentRecordFromJSON(data []byte) (TorrentRecord, error) {
	var t TorrentRecord
	if err := json.Unmarshal(data, &t); err != nil {
		return TorrentRecord{}, fmt.Errorf("TorrentRecordFromJSON: %w", err)
	}
	return t, nil
}

type TorrentRecord struct {
	CollectionID   string    `json:"collection_id"`
	CollectionName string    `json:"collection_name"`
	CreatedAt      time.Time `json:"created_at"`
	IsDualAudio    bool      `json:"is_dual_audio"`
	Files          []File    `json:"files"`
	ID             string    `json:"id"`
	Infohash       *string   `json:"infohash"`
	IsBest         bool      `json:"is_best"`
	ReleaseGroup   string    `json:"release_group"`
	Tags           []Tag     `json:"tags"`
	Tracker        Tracker   `json:"tracker"`
	UpdatedAt      time.Time `json:"updated_at"`
	URL            string    `json:"url"`
	GroupedURL     *string   `json:"grouped_url"`
	Size           int64     `json:"size"`
}

type torrentRecordAPI struct {
	CollectionID   string `json:"collectionId"`
	CollectionName string `json:"collectionName"`
	Created        string `json:"created"`
	DualAudio      bool   `json:"dualAudio"`
	Files          []struct {
		Length int64  `json:"length"`
		Name   string `json:"name"`
	} `json:"files"`
	GroupedURL   string   `json:"groupedUrl"`
	ID           string   `json:"id"`
	InfoHash     string   `json:"infoHash"`
	IsBest       bool     `json:"isBest"`
	ReleaseGroup string   `json:"releaseGroup"`
	Tags         []string `json:"tags"`
	Tracker      string   `json:"tracker"`
	Updated      string   `json:"updated"`
	URL          string   `json:"url"`
}

func torrentRecordFromAPI(raw torrentRecordAPI) (TorrentRecord, error) {
	createdAt, err := parseSeaDexTime(raw.Created)
	if err != nil {
		return TorrentRecord{}, fmt.Errorf("parsing created_at: %w", err)
	}
	updatedAt, err := parseSeaDexTime(raw.Updated)
	if err != nil {
		return TorrentRecord{}, fmt.Errorf("parsing updated_at: %w", err)
	}

	tracker, err := ParseTracker(raw.Tracker)
	if err != nil {
		return TorrentRecord{}, fmt.Errorf("parsing tracker: %w", err)
	}

	files := make([]File, 0, len(raw.Files))
	var totalSize int64
	for _, f := range raw.Files {
		files = append(files, File{Name: f.Name, Size: f.Length})
		totalSize += f.Length
	}

	tags := make([]Tag, 0, len(raw.Tags))
	for _, t := range raw.Tags {
		tag, err := ParseTag(t)
		if err != nil {
			return TorrentRecord{}, fmt.Errorf("parsing tag %q: %w", t, err)
		}
		tags = append(tags, tag)
	}

	var infohash *string
	if raw.InfoHash != "<redacted>" && raw.InfoHash != "" {
		h := raw.InfoHash
		infohash = &h
	}

	var groupedURL *string
	if raw.GroupedURL != "" {
		g := raw.GroupedURL
		groupedURL = &g
	}

	torrentURL := raw.URL
	if tracker.IsPrivate() && !strings.HasPrefix(torrentURL, "http") {
		base := tracker.URL()
		if base != "" {
			u, err := url.Parse(base)
			if err == nil {
				ref, err := url.Parse(torrentURL)
				if err == nil {
					torrentURL = u.ResolveReference(ref).String()
				}
			}
		}
	}

	return TorrentRecord{
		CollectionID:   raw.CollectionID,
		CollectionName: raw.CollectionName,
		CreatedAt:      createdAt,
		IsDualAudio:    raw.DualAudio,
		Files:          files,
		ID:             raw.ID,
		Infohash:       infohash,
		IsBest:         raw.IsBest,
		ReleaseGroup:   raw.ReleaseGroup,
		Tags:           tags,
		Tracker:        tracker,
		UpdatedAt:      updatedAt,
		URL:            torrentURL,
		GroupedURL:     groupedURL,
		Size:           totalSize,
	}, nil
}

func (e EntryRecord) ToDict() map[string]any {
	torrents := make([]map[string]any, len(e.Torrents))
	for i, t := range e.Torrents {
		torrents[i] = t.ToDict()
	}
	var theoreticalBest any
	if e.TheoreticalBest != nil {
		theoreticalBest = *e.TheoreticalBest
	}
	return map[string]any{
		"anilist_id":       e.AnilistID,
		"collection_id":    e.CollectionID,
		"collection_name":  e.CollectionName,
		"comparisons":      e.Comparisons,
		"created_at":       e.CreatedAt,
		"id":               e.ID,
		"is_incomplete":    e.IsIncomplete,
		"notes":            e.Notes,
		"theoretical_best": theoreticalBest,
		"torrents":         torrents,
		"updated_at":       e.UpdatedAt,
		"url":              e.URL,
		"size":             e.Size,
	}
}

func (e EntryRecord) ToJSON() (string, error) {
	b, err := marshalIndent(e)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func EntryRecordFromJSON(data []byte) (EntryRecord, error) {
	var e EntryRecord
	if err := json.Unmarshal(data, &e); err != nil {
		return EntryRecord{}, fmt.Errorf("EntryRecordFromJSON: %w", err)
	}
	return e, nil
}

type EntryRecord struct {
	AnilistID       int             `json:"anilist_id"`
	CollectionID    string          `json:"collection_id"`
	CollectionName  string          `json:"collection_name"`
	Comparisons     []string        `json:"comparisons"`
	CreatedAt       time.Time       `json:"created_at"`
	ID              string          `json:"id"`
	IsIncomplete    bool            `json:"is_incomplete"`
	Notes           string          `json:"notes"`
	TheoreticalBest *string         `json:"theoretical_best"`
	Torrents        []TorrentRecord `json:"torrents"`
	UpdatedAt       time.Time       `json:"updated_at"`
	URL             string          `json:"url"`
	Size            int64           `json:"size"`
}

type entryRecordAPI struct {
	AlID           int    `json:"alID"`
	CollectionID   string `json:"collectionId"`
	CollectionName string `json:"collectionName"`
	Comparison     string `json:"comparison"`
	Created        string `json:"created"`
	Expand         struct {
		Trs []torrentRecordAPI `json:"trs"`
	} `json:"expand"`
	ID              string `json:"id"`
	Incomplete      bool   `json:"incomplete"`
	Notes           string `json:"notes"`
	TheoreticalBest string `json:"theoreticalBest"`
	Updated         string `json:"updated"`
}

func entryRecordFromAPI(raw entryRecordAPI) (EntryRecord, error) {
	createdAt, err := parseSeaDexTime(raw.Created)
	if err != nil {
		return EntryRecord{}, fmt.Errorf("parsing created_at: %w", err)
	}
	updatedAt, err := parseSeaDexTime(raw.Updated)
	if err != nil {
		return EntryRecord{}, fmt.Errorf("parsing updated_at: %w", err)
	}

	torrents := make([]TorrentRecord, 0, len(raw.Expand.Trs))
	var totalSize int64
	for _, tr := range raw.Expand.Trs {
		t, err := torrentRecordFromAPI(tr)
		if err != nil {
			return EntryRecord{}, fmt.Errorf("parsing torrent %s: %w", tr.ID, err)
		}
		torrents = append(torrents, t)
		totalSize += t.Size
	}

	var comparisons []string
	for _, c := range strings.Split(raw.Comparison, ",") {
		c = strings.TrimSpace(c)
		if c != "" {
			comparisons = append(comparisons, c)
		}
	}

	var theoreticalBest *string
	if raw.TheoreticalBest != "" {
		tb := raw.TheoreticalBest
		theoreticalBest = &tb
	}

	return EntryRecord{
		AnilistID:       raw.AlID,
		CollectionID:    raw.CollectionID,
		CollectionName:  raw.CollectionName,
		Comparisons:     comparisons,
		CreatedAt:       createdAt,
		ID:              raw.ID,
		IsIncomplete:    raw.Incomplete,
		Notes:           raw.Notes,
		TheoreticalBest: theoreticalBest,
		Torrents:        torrents,
		UpdatedAt:       updatedAt,
		URL:             fmt.Sprintf("https://releases.moe/%d/", raw.AlID),
		Size:            totalSize,
	}, nil
}

func marshalIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

type listResponse struct {
	Page       int               `json:"page"`
	PerPage    int               `json:"perPage"`
	TotalItems int               `json:"totalItems"`
	TotalPages int               `json:"totalPages"`
	Items      []json.RawMessage `json:"items"`
}

func parseSeaDexTime(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04:05.000Z",
		"2006-01-02 15:04:05.999Z",
		"2006-01-02 15:04:05Z",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}
