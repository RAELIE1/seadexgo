package seadex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	seadex "github.com/RAELIE1/seadexgo"
)

func loadSampleResponse(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("../seadex/tests/sample_response.json")
	if err != nil {
		t.Fatalf("reading sample_response.json: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parsing sample_response.json: %v", err)
	}
	return m
}

func newMockServer(t *testing.T, sampleResponse map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/graphql" || r.Host == "graphql.anilist.co" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"Media": map[string]any{
						"id": 165790,
						"title": map[string]any{
							"english": "365 Days to the Wedding",
							"romaji":  "Kekkon Suru tte, Hontou desu ka",
						},
					},
				},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleResponse)
	}))
}

func TestTrackerValues(t *testing.T) {
	cases := []struct {
		tracker seadex.Tracker
		value   string
	}{
		{seadex.TrackerNyaa, "Nyaa"},
		{seadex.TrackerAnimeTosho, "AnimeTosho"},
		{seadex.TrackerAniDex, "AniDex"},
		{seadex.TrackerRuTracker, "RuTracker"},
		{seadex.TrackerAnimeBytes, "AB"},
		{seadex.TrackerBeyondHD, "BeyondHD"},
		{seadex.TrackerPassThePopcorn, "PassThePopcorn"},
		{seadex.TrackerBroadcastTheNet, "BroadcastTheNet"},
		{seadex.TrackerHDBits, "HDBits"},
		{seadex.TrackerBlutopia, "Blutopia"},
		{seadex.TrackerAither, "Aither"},
		{seadex.TrackerOther, "Other"},
		{seadex.TrackerOtherPrivate, "OtherPrivate"},
	}
	for _, c := range cases {
		if string(c.tracker) != c.value {
			t.Errorf("Tracker %v: got %q, want %q", c.tracker, string(c.tracker), c.value)
		}
	}
}

func TestTrackerIsPublicPrivate(t *testing.T) {
	cases := []struct {
		tracker   string
		isPrivate bool
		isPublic  bool
	}{
		{"Nyaa", false, true},
		{"AnimeTosho", false, true},
		{"AniDex", false, true},
		{"RuTracker", false, true},
		{"AB", true, false},
		{"BeyondHD", true, false},
		{"PassThePopcorn", true, false},
		{"BroadcastTheNet", true, false},
		{"HDBits", true, false},
		{"Blutopia", true, false},
		{"Aither", true, false},
		{"Other", false, true},
		{"OtherPrivate", true, false},
	}
	for _, c := range cases {
		tr, err := seadex.ParseTracker(c.tracker)
		if err != nil {
			t.Fatalf("ParseTracker(%q): %v", c.tracker, err)
		}
		if tr.IsPrivate() != c.isPrivate {
			t.Errorf("Tracker(%q).IsPrivate(): got %v, want %v", c.tracker, tr.IsPrivate(), c.isPrivate)
		}
		if tr.IsPublic() != c.isPublic {
			t.Errorf("Tracker(%q).IsPublic(): got %v, want %v", c.tracker, tr.IsPublic(), c.isPublic)
		}
	}
}

func TestTrackerURL(t *testing.T) {
	cases := []struct{ tracker, wantURL string }{
		{"Nyaa", "https://nyaa.si"},
		{"AnimeTosho", "https://animetosho.org"},
		{"AniDex", "https://anidex.info"},
		{"RuTracker", "https://rutracker.org"},
		{"AB", "https://animebytes.tv"},
		{"BeyondHD", "https://beyond-hd.me"},
		{"PassThePopcorn", "https://passthepopcorn.me"},
		{"BroadcastTheNet", "https://broadcasthe.net"},
		{"HDBits", "https://hdbits.org"},
		{"Blutopia", "https://blutopia.cc"},
		{"Aither", "https://aither.cc"},
		{"Other", ""},
		{"OtherPrivate", ""},
	}
	for _, c := range cases {
		tr, err := seadex.ParseTracker(c.tracker)
		if err != nil {
			t.Fatalf("ParseTracker(%q): %v", c.tracker, err)
		}
		if tr.URL() != c.wantURL {
			t.Errorf("Tracker(%q).URL(): got %q, want %q", c.tracker, tr.URL(), c.wantURL)
		}
	}
}

func TestTrackerCaseInsensitive(t *testing.T) {
	cases := []struct {
		input string
		want  seadex.Tracker
	}{
		{"nyAA", seadex.TrackerNyaa},
		{"ANIMETOSHO", seadex.TrackerAnimeTosho},
		{"Ab", seadex.TrackerAnimeBytes},
	}
	for _, c := range cases {
		got, err := seadex.ParseTracker(c.input)
		if err != nil {
			t.Fatalf("ParseTracker(%q): %v", c.input, err)
		}
		if got != c.want {
			t.Errorf("ParseTracker(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestTrackerBadValue(t *testing.T) {
	_, err := seadex.ParseTracker("notavalidtracker")
	if err == nil {
		t.Error("expected error for invalid tracker, got nil")
	}
}

func TestErrors(t *testing.T) {
	var err error

	err = &seadex.SeaDexError{Message: "base error"}
	if err.Error() != "base error" {
		t.Errorf("SeaDexError.Error() = %q", err.Error())
	}

	err = &seadex.EntryNotFoundError{SeaDexError: seadex.SeaDexError{Message: "not found"}}
	if err.Error() != "not found" {
		t.Errorf("EntryNotFoundError.Error() = %q", err.Error())
	}

	err = &seadex.BadBackupFileError{SeaDexError: seadex.SeaDexError{Message: "bad file"}}
	if err.Error() != "bad file" {
		t.Errorf("BadBackupFileError.Error() = %q", err.Error())
	}
}

func TestEntryRecord(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(
		seadex.WithBaseURL(srv.URL),
		seadex.WithHTTPClient(srv.Client()),
	)

	entry, err := client.FromID(165790)
	if err != nil {
		t.Fatalf("FromID: %v", err)
	}

	if entry.AnilistID != 165790 {
		t.Errorf("AnilistID = %d, want 165790", entry.AnilistID)
	}
	if entry.CollectionID != "3l2x9nxip35gqb5" {
		t.Errorf("CollectionID = %q", entry.CollectionID)
	}
	if entry.CollectionName != "entries" {
		t.Errorf("CollectionName = %q", entry.CollectionName)
	}
	if len(entry.Comparisons) != 1 || entry.Comparisons[0] != "https://slow.pics/c/ntpJn04T" {
		t.Errorf("Comparisons = %v", entry.Comparisons)
	}
	wantCreated := time.Date(2025, 3, 5, 22, 27, 18, 283_000_000, time.UTC)
	if !entry.CreatedAt.Equal(wantCreated) {
		t.Errorf("CreatedAt = %v, want %v", entry.CreatedAt, wantCreated)
	}
	if entry.ID != "ydydj1p7bn3o7ro" {
		t.Errorf("ID = %q", entry.ID)
	}
	if entry.IsIncomplete {
		t.Error("IsIncomplete should be false")
	}
	if entry.TheoreticalBest != nil {
		t.Errorf("TheoreticalBest should be nil, got %v", entry.TheoreticalBest)
	}
	wantUpdated := time.Date(2025, 8, 1, 22, 48, 15, 341_000_000, time.UTC)
	if !entry.UpdatedAt.Equal(wantUpdated) {
		t.Errorf("UpdatedAt = %v, want %v", entry.UpdatedAt, wantUpdated)
	}
	if entry.URL != "https://releases.moe/165790/" {
		t.Errorf("URL = %q", entry.URL)
	}
	if entry.Size != 119397238820 {
		t.Errorf("Size = %d, want 119397238820", entry.Size)
	}
	if len(entry.Torrents) != 14 {
		t.Errorf("len(Torrents) = %d, want 14", len(entry.Torrents))
	}

	tr := entry.Torrents[0]
	if tr.ID != "z2hmkedvvo6z9la" {
		t.Errorf("Torrents[0].ID = %q", tr.ID)
	}
	if tr.ReleaseGroup != "-ZR-" {
		t.Errorf("Torrents[0].ReleaseGroup = %q", tr.ReleaseGroup)
	}
	if tr.Tracker != seadex.TrackerAnimeBytes {
		t.Errorf("Torrents[0].Tracker = %v", tr.Tracker)
	}
	if !tr.IsBest {
		t.Error("Torrents[0].IsBest should be true")
	}
	if tr.Infohash != nil {
		t.Errorf("Torrents[0].Infohash should be nil (redacted), got %v", tr.Infohash)
	}
	if tr.GroupedURL != nil {
		t.Errorf("Torrents[0].GroupedURL should be nil, got %v", tr.GroupedURL)
	}
	if len(tr.Files) != 14 {
		t.Errorf("Torrents[0] file count = %d, want 14", len(tr.Files))
	}
	if tr.Size != 84778716790 {
		t.Errorf("Torrents[0].Size = %d, want 84778716790", tr.Size)
	}
	if tr.URL != "https://animebytes.tv/torrents.php?id=94644&torrentid=1160250" {
		t.Errorf("Torrents[0].URL = %q", tr.URL)
	}

	nyaa := entry.Torrents[2]
	if nyaa.Tracker != seadex.TrackerNyaa {
		t.Errorf("Torrents[2].Tracker = %v", nyaa.Tracker)
	}
	if nyaa.Infohash == nil || *nyaa.Infohash != "c4c1031570089d70bff40e1a89253025ad1cead7" {
		t.Errorf("Torrents[2].Infohash = %v", nyaa.Infohash)
	}
	if nyaa.GroupedURL == nil {
		t.Error("Torrents[2].GroupedURL should not be nil")
	}
}

func TestSeaDexEntryBaseURL(t *testing.T) {
	client := seadex.NewSeaDexEntry()
	if client.BaseURL() != "https://releases.moe" {
		t.Errorf("BaseURL() = %q", client.BaseURL())
	}
}

func TestFromAnilistID(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entry, err := client.FromID(165790)
	if err != nil {
		t.Fatalf("FromID(int): %v", err)
	}
	if entry.AnilistID != 165790 {
		t.Errorf("AnilistID = %d", entry.AnilistID)
	}
}

func TestFromSeaDexID(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entry, err := client.FromID("ydydj1p7bn3o7ro")
	if err != nil {
		t.Fatalf("FromID(string): %v", err)
	}
	if entry.AnilistID != 165790 {
		t.Errorf("AnilistID = %d", entry.AnilistID)
	}
}

func TestFromFilename(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entries, err := client.FromFilename("[SubsPlease] Kekkon suru tte, Hontou desu ka - 01 (1080p) [29AE676E].mkv")
	if err != nil {
		t.Fatalf("FromFilename: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	if entries[0].AnilistID != 165790 {
		t.Errorf("AnilistID = %d", entries[0].AnilistID)
	}
}

func TestFromInfohash(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entries, err := client.FromInfohash("c4c1031570089d70bff40e1a89253025ad1cead7")
	if err != nil {
		t.Fatalf("FromInfohash: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].AnilistID != 165790 {
		t.Errorf("AnilistID = %d", entries[0].AnilistID)
	}
}

func TestFromInfohashInvalid(t *testing.T) {
	client := seadex.NewSeaDexEntry()
	_, err := client.FromInfohash("notahash")
	if err == nil {
		t.Error("expected error for invalid infohash")
	}
}

func TestFromFilter(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entries, err := client.FromFilter("alID=165790")
	if err != nil {
		t.Fatalf("FromFilter: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries")
	}
}

func TestIterator(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entries, err := client.Iterator()
	if err != nil {
		t.Fatalf("Iterator: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries from iterator")
	}
}

func TestFromTitle(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))

	entry, err := client.FromTitle("365 days to the wedding")
	if err != nil {
		t.Fatalf("FromTitle (first call): %v", err)
	}
	if entry.AnilistID != 165790 {
		t.Errorf("AnilistID = %d, want 165790", entry.AnilistID)
	}

	title := client.AnilistTitle("365 days to the wedding")
	if title == "" {
		t.Error("AnilistTitle returned empty string after FromTitle")
	}

	entry2, err := client.FromTitle("365 days to the wedding")
	if err != nil {
		t.Fatalf("FromTitle (cached): %v", err)
	}
	if entry2.AnilistID != entry.AnilistID {
		t.Errorf("cached AnilistID = %d, want %d", entry2.AnilistID, entry.AnilistID)
	}
}

func TestAnilistTitleMiss(t *testing.T) {
	client := seadex.NewSeaDexEntry()
	if got := client.AnilistTitle("not looked up yet"); got != "" {
		t.Errorf("AnilistTitle for unknown term = %q, want empty", got)
	}
}

func TestFromFilterStream(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))

	stream, err := client.FromFilterStream("alID=165790")
	if err != nil {
		t.Fatalf("FromFilterStream: %v", err)
	}

	var count int
	for stream.Next() {
		_ = stream.Value()
		count++
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one entry from FromFilterStream")
	}
}

func TestFromFilterStreamEmptyFilter(t *testing.T) {
	client := seadex.NewSeaDexEntry()
	_, err := client.FromFilterStream("")
	if err == nil {
		t.Error("expected error for empty filter")
	}
}

func TestIteratorStream(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))

	stream := client.IteratorStream()
	var count int
	for stream.Next() {
		_ = stream.Value()
		count++
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if count == 0 {
		t.Error("expected entries from IteratorStream")
	}
}

func TestFromFilenameStream(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))

	stream := client.FromFilenameStream("[SubsPlease] Kekkon suru tte, Hontou desu ka - 01 (1080p) [29AE676E].mkv")
	var found bool
	for stream.Next() {
		if stream.Value().AnilistID == 165790 {
			found = true
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if !found {
		t.Error("expected entry with AnilistID 165790 from FromFilenameStream")
	}
}

func TestFromInfohashStream(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))

	stream, err := client.FromInfohashStream("c4c1031570089d70bff40e1a89253025ad1cead7")
	if err != nil {
		t.Fatalf("FromInfohashStream: %v", err)
	}
	var count int
	for stream.Next() {
		count++
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 result, got %d", count)
	}
}

func TestFromInfohashStreamInvalid(t *testing.T) {
	client := seadex.NewSeaDexEntry()
	_, err := client.FromInfohashStream("notahash")
	if err == nil {
		t.Error("expected error for invalid infohash")
	}
}

func TestEntryRecordToJSONRoundtrip(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entry, err := client.FromID(165790)
	if err != nil {
		t.Fatalf("FromID: %v", err)
	}

	jsonStr, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if len(jsonStr) == 0 {
		t.Fatal("ToJSON returned empty string")
	}

	restored, err := seadex.EntryRecordFromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("EntryRecordFromJSON: %v", err)
	}
	if restored.AnilistID != entry.AnilistID {
		t.Errorf("round-trip AnilistID = %d, want %d", restored.AnilistID, entry.AnilistID)
	}
	if restored.ID != entry.ID {
		t.Errorf("round-trip ID = %q, want %q", restored.ID, entry.ID)
	}
	if len(restored.Torrents) != len(entry.Torrents) {
		t.Errorf("round-trip Torrents len = %d, want %d", len(restored.Torrents), len(entry.Torrents))
	}
	if restored.Size != entry.Size {
		t.Errorf("round-trip Size = %d, want %d", restored.Size, entry.Size)
	}
}

func TestTorrentRecordToJSONRoundtrip(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entry, err := client.FromID(165790)
	if err != nil {
		t.Fatalf("FromID: %v", err)
	}

	tr := entry.Torrents[0]
	jsonStr, err := tr.ToJSON()
	if err != nil {
		t.Fatalf("TorrentRecord.ToJSON: %v", err)
	}

	restored, err := seadex.TorrentRecordFromJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("TorrentRecordFromJSON: %v", err)
	}
	if restored.ID != tr.ID {
		t.Errorf("round-trip ID = %q, want %q", restored.ID, tr.ID)
	}
	if restored.Size != tr.Size {
		t.Errorf("round-trip Size = %d, want %d", restored.Size, tr.Size)
	}
}

func TestEntryRecordToDict(t *testing.T) {
	sample := loadSampleResponse(t)
	srv := newMockServer(t, sample)
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	entry, err := client.FromID(165790)
	if err != nil {
		t.Fatalf("FromID: %v", err)
	}

	d := entry.ToDict()
	if d["anilist_id"] != 165790 {
		t.Errorf("ToDict anilist_id = %v", d["anilist_id"])
	}
	if d["id"] != "ydydj1p7bn3o7ro" {
		t.Errorf("ToDict id = %v", d["id"])
	}
	torrents, ok := d["torrents"].([]map[string]any)
	if !ok {
		t.Fatalf("ToDict torrents wrong type: %T", d["torrents"])
	}
	if len(torrents) != 14 {
		t.Errorf("ToDict torrents len = %d, want 14", len(torrents))
	}
}

func TestEntryNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"page": 1, "perPage": 500, "totalItems": 0, "totalPages": 1, "items": []any{},
		})
	}))
	defer srv.Close()

	client := seadex.NewSeaDexEntry(seadex.WithBaseURL(srv.URL), seadex.WithHTTPClient(srv.Client()))
	_, err := client.FromID(999999)
	if err == nil {
		t.Fatal("expected EntryNotFoundError")
	}
	var nfe *seadex.EntryNotFoundError
	if _, ok := err.(*seadex.EntryNotFoundError); !ok {
		_ = nfe
		t.Errorf("expected *EntryNotFoundError, got %T", err)
	}
}
