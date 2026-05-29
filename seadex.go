package seadex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://releases.moe"
	version        = "0.7.2"
)

type SeaDexEntry struct {
	baseURL      string
	endpoint     string
	client       *http.Client
	cacheMu      sync.RWMutex
	anilistCache map[string]anilistCacheEntry
}

type anilistCacheEntry struct {
	id      int
	english string
	romaji  string
}

func NewSeaDexEntry(opts ...func(*SeaDexEntry)) *SeaDexEntry {
	s := &SeaDexEntry{
		baseURL:      defaultBaseURL,
		endpoint:     defaultBaseURL + "/api/collections/entries/records",
		client:       defaultHTTPClient(),
		anilistCache: make(map[string]anilistCacheEntry),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func WithBaseURL(baseURL string) func(*SeaDexEntry) {
	return func(s *SeaDexEntry) {
		s.baseURL = strings.TrimRight(baseURL, "/")
		s.endpoint = s.baseURL + "/api/collections/entries/records"
	}
}

func WithHTTPClient(client *http.Client) func(*SeaDexEntry) {
	return func(s *SeaDexEntry) {
		s.client = client
	}
}

func (s *SeaDexEntry) BaseURL() string {
	return s.baseURL
}

func (s *SeaDexEntry) Close() {}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Minute,
		Transport: &userAgentTransport{
			ua:   fmt.Sprintf("seadex/%s (https://github.com/darkNatsumi/seadex)", version),
			base: http.DefaultTransport,
		},
	}
}

type userAgentTransport struct {
	ua   string
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", t.ua)
	return t.base.RoundTrip(req)
}

func (s *SeaDexEntry) fetchPage(ctx context.Context, params url.Values, page int) (*listResponse, error) {
	p := url.Values{}
	for k, v := range params {
		p[k] = v
	}
	if page > 1 {
		p.Set("page", strconv.Itoa(page))
	}
	reqURL := s.endpoint + "?" + p.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SeaDex API returned %d", resp.StatusCode)
	}
	var lr listResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, err
	}
	return &lr, nil
}

func (s *SeaDexEntry) fromFilter(filter string, paginate bool) ([]EntryRecord, error) {
	return s.fromFilterCtx(context.Background(), filter, paginate)
}

func (s *SeaDexEntry) fromFilterCtx(ctx context.Context, filter string, paginate bool) ([]EntryRecord, error) {
	params := url.Values{}
	params.Set("perPage", "500")
	params.Set("expand", "trs")
	if filter != "" {
		params.Set("filter", filter)
	}
	if !paginate {
		params.Set("skipTotal", "true")
	}

	lr, err := s.fetchPage(ctx, params, 1)
	if err != nil {
		return nil, err
	}

	all, err := parseItems(lr.Items)
	if err != nil {
		return nil, err
	}

	if paginate && lr.TotalPages > 1 {
		type result struct {
			lr  *listResponse
			err error
		}

		nextCh := make(chan result, 1)
		go func() {
			pg, e := s.fetchPage(ctx, params, 2)
			nextCh <- result{pg, e}
		}()

		for page := 2; page <= lr.TotalPages; page++ {
			res := <-nextCh
			if res.err != nil {
				return nil, res.err
			}
			if page < lr.TotalPages {
				next := page + 1
				go func() {
					pg, e := s.fetchPage(ctx, params, next)
					nextCh <- result{pg, e}
				}()
			}
			records, err := parseItems(res.lr.Items)
			if err != nil {
				return nil, err
			}
			all = append(all, records...)
		}
	}

	return all, nil
}

func parseItems(items []json.RawMessage) ([]EntryRecord, error) {
	records := make([]EntryRecord, 0, len(items))
	for _, raw := range items {
		var api entryRecordAPI
		if err := json.Unmarshal(raw, &api); err != nil {
			return nil, err
		}
		rec, err := entryRecordFromAPI(api)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func (s *SeaDexEntry) FromFilter(filter string) ([]EntryRecord, error) {
	return s.FromFilterContext(context.Background(), filter)
}

func (s *SeaDexEntry) FromFilterContext(ctx context.Context, filter string) ([]EntryRecord, error) {
	if filter == "" {
		return nil, fmt.Errorf("filter must not be empty")
	}
	return s.fromFilterCtx(ctx, filter, true)
}

func (s *SeaDexEntry) FromID(id any) (EntryRecord, error) {
	return s.FromIDContext(context.Background(), id)
}

func (s *SeaDexEntry) FromIDContext(ctx context.Context, id any) (EntryRecord, error) {
	var filter string
	switch v := id.(type) {
	case int:
		filter = fmt.Sprintf("alID=%d", v)
	case string:
		filter = fmt.Sprintf("id='%s'", v)
	default:
		return EntryRecord{}, fmt.Errorf("id must be int or string, got %T", id)
	}

	entries, err := s.fromFilterCtx(ctx, filter, false)
	if err != nil {
		return EntryRecord{}, err
	}
	if len(entries) == 0 {
		return EntryRecord{}, newEntryNotFoundError("no seadex entry found for id: %v", id)
	}
	return entries[0], nil
}

func (s *SeaDexEntry) FromTitle(title string) (EntryRecord, error) {
	return s.FromTitleContext(context.Background(), title)
}

func (s *SeaDexEntry) FromTitleContext(ctx context.Context, title string) (EntryRecord, error) {
	title = strings.TrimSpace(title)
	anilistID, err := s.anilistIDFromTitleCtx(ctx, title)
	if err != nil {
		return EntryRecord{}, newEntryNotFoundError("no seadex entry found for title: %s", title)
	}

	entries, err := s.fromFilterCtx(ctx, fmt.Sprintf("alID=%d", anilistID), false)
	if err != nil || len(entries) == 0 {
		return EntryRecord{}, newEntryNotFoundError("no seadex entry found for title: %s", title)
	}
	return entries[0], nil
}

func (s *SeaDexEntry) anilistIDFromTitleCtx(ctx context.Context, title string) (int, error) {
	s.cacheMu.RLock()
	cached, ok := s.anilistCache[title]
	s.cacheMu.RUnlock()
	if ok {
		return cached.id, nil
	}

	query := `query ($search: String!) { Media(search: $search, type: ANIME) { id title { english romaji } } }`
	payload := map[string]any{
		"query":     query,
		"variables": map[string]any{"search": title},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graphql.anilist.co", strings.NewReader(string(body)))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Media *struct {
				ID    int `json:"id"`
				Title struct {
					English string `json:"english"`
					Romaji  string `json:"romaji"`
				} `json:"title"`
			} `json:"Media"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if result.Data.Media == nil {
		return 0, fmt.Errorf("no AniList entry found for title: %s", title)
	}

	s.cacheMu.Lock()
	s.anilistCache[title] = anilistCacheEntry{
		id:      result.Data.Media.ID,
		english: result.Data.Media.Title.English,
		romaji:  result.Data.Media.Title.Romaji,
	}
	s.cacheMu.Unlock()
	return result.Data.Media.ID, nil
}

func (s *SeaDexEntry) AnilistTitle(searchTerm string) string {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if cached, ok := s.anilistCache[searchTerm]; ok {
		if cached.english != "" {
			return cached.english
		}
		return cached.romaji
	}
	return ""
}

type EntryStream struct {
	ch       <-chan EntryRecord
	err      <-chan error
	cur      EntryRecord
	done     bool
	finalErr error
}

func (s *EntryStream) Next() bool {
	if s.done {
		return false
	}
	v, ok := <-s.ch
	if !ok {
		s.finalErr = <-s.err
		s.done = true
		return false
	}
	s.cur = v
	return true
}

func (s *EntryStream) Value() EntryRecord { return s.cur }

func (s *EntryStream) Err() error { return s.finalErr }

func newEntryStream(ch <-chan EntryRecord, errCh <-chan error) *EntryStream {
	return &EntryStream{ch: ch, err: errCh}
}

func (s *SeaDexEntry) streamFilterCtx(ctx context.Context, filter string, paginate bool) *EntryStream {
	ch := make(chan EntryRecord, 32)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)

		params := url.Values{}
		params.Set("perPage", "500")
		params.Set("expand", "trs")
		if filter != "" {
			params.Set("filter", filter)
		}
		if !paginate {
			params.Set("skipTotal", "true")
		}

		sendItems := func(items []json.RawMessage) error {
			for _, raw := range items {
				var api entryRecordAPI
				if err := json.Unmarshal(raw, &api); err != nil {
					return err
				}
				rec, err := entryRecordFromAPI(api)
				if err != nil {
					return err
				}
				select {
				case ch <- rec:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}

		lr, err := s.fetchPage(ctx, params, 1)
		if err != nil {
			errCh <- err
			return
		}
		if err := sendItems(lr.Items); err != nil {
			errCh <- err
			return
		}

		if paginate && lr.TotalPages > 1 {
			type result struct {
				lr  *listResponse
				err error
			}
			nextCh := make(chan result, 1)
			go func() {
				pg, e := s.fetchPage(ctx, params, 2)
				nextCh <- result{pg, e}
			}()

			for page := 2; page <= lr.TotalPages; page++ {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case res := <-nextCh:
					if res.err != nil {
						errCh <- res.err
						return
					}
					if page < lr.TotalPages {
						next := page + 1
						go func() {
							pg, e := s.fetchPage(ctx, params, next)
							nextCh <- result{pg, e}
						}()
					}
					if err := sendItems(res.lr.Items); err != nil {
						errCh <- err
						return
					}
				}
			}
		}

		errCh <- nil
	}()

	return newEntryStream(ch, errCh)
}

func (s *SeaDexEntry) streamFilter(filter string, paginate bool) *EntryStream {
	return s.streamFilterCtx(context.Background(), filter, paginate)
}

func (s *SeaDexEntry) FromFilterStream(filter string) (*EntryStream, error) {
	if filter == "" {
		return nil, fmt.Errorf("filter must not be empty")
	}
	return s.streamFilter(filter, true), nil
}

func (s *SeaDexEntry) FromFilenameStream(filename string) *EntryStream {
	base := path.Base(filename)
	filter := fmt.Sprintf(`trs.files?~'"name":"%s"'`, base)
	return s.streamFilter(filter, false)
}

func (s *SeaDexEntry) FromInfohashStream(infohash string) (*EntryStream, error) {
	infohash = strings.ToLower(strings.TrimSpace(infohash))
	if !infohashPattern.MatchString(infohash) {
		return nil, fmt.Errorf("invalid infohash format: must be a 40-character hexadecimal string")
	}
	filter := fmt.Sprintf("trs.infoHash?='%s'", infohash)
	return s.streamFilter(filter, false), nil
}

func (s *SeaDexEntry) IteratorStream() *EntryStream {
	return s.streamFilter("", true)
}

func (s *SeaDexEntry) FromFilename(filename string) ([]EntryRecord, error) {
	base := path.Base(filename)
	filter := fmt.Sprintf(`trs.files?~'"name":"%s"'`, base)
	return s.fromFilter(filter, false)
}

var infohashPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func (s *SeaDexEntry) FromInfohash(infohash string) ([]EntryRecord, error) {
	infohash = strings.ToLower(strings.TrimSpace(infohash))
	if !infohashPattern.MatchString(infohash) {
		return nil, fmt.Errorf("invalid infohash format: must be a 40-character hexadecimal string")
	}
	filter := fmt.Sprintf("trs.infoHash?='%s'", infohash)
	return s.fromFilter(filter, false)
}

func (s *SeaDexEntry) Iterator() ([]EntryRecord, error) {
	return s.fromFilter("", true)
}

func Entries() ([]EntryRecord, error) {
	client := NewSeaDexEntry()
	defer client.Close()
	return client.Iterator()
}
