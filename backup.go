package seadex

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type BackupFile struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
}

func (b BackupFile) String() string {
	return b.Name
}

type backupFileAPI struct {
	Key      string `json:"key"`
	Modified string `json:"modified"`
	Size     int64  `json:"size"`
}

func backupFileFromAPI(raw backupFileAPI) (BackupFile, error) {
	t, err := parseSeaDexTime(raw.Modified)
	if err != nil {
		return BackupFile{}, fmt.Errorf("parsing modified_time: %w", err)
	}
	return BackupFile{Name: raw.Key, Size: raw.Size, ModifiedTime: t}, nil
}

type SeaDexBackup struct {
	baseURL    string
	client     *http.Client
	adminToken string
}

func NewSeaDexBackup(email, password string, opts ...func(*SeaDexBackup)) (*SeaDexBackup, error) {
	b := &SeaDexBackup{
		baseURL: defaultBaseURL,
		client:  defaultHTTPClient(),
	}
	for _, o := range opts {
		o(b)
	}

	token, err := b.authWithPassword(email, password)
	if err != nil {
		return nil, fmt.Errorf("SeaDexBackup auth failed: %w", err)
	}
	b.adminToken = token
	return b, nil
}

func WithBackupBaseURL(baseURL string) func(*SeaDexBackup) {
	return func(b *SeaDexBackup) {
		b.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithBackupHTTPClient(client *http.Client) func(*SeaDexBackup) {
	return func(b *SeaDexBackup) {
		b.client = client
	}
}

func (b *SeaDexBackup) BaseURL() string {
	return b.baseURL
}

func (b *SeaDexBackup) Close() {}

func (b *SeaDexBackup) urlFor(endpoint string) string {
	return b.baseURL + endpoint
}

func (b *SeaDexBackup) authWithPassword(email, password string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"identity": email, "password": password})
	req, err := http.NewRequest(http.MethodPost, b.urlFor("/api/admins/auth-with-password"), strings.NewReader(string(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth returned %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (b *SeaDexBackup) getFileToken() (string, error) {
	req, err := http.NewRequest(http.MethodPost, b.urlFor("/api/files/token"), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", b.adminToken)

	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file token returned %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (b *SeaDexBackup) GetBackups() ([]BackupFile, error) {
	req, err := http.NewRequest(http.MethodGet, b.urlFor("/api/backups"), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", b.adminToken)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []backupFileAPI
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	backups := make([]BackupFile, 0, len(raw))
	for _, r := range raw {
		bf, err := backupFileFromAPI(r)
		if err != nil {
			return nil, err
		}
		backups = append(backups, bf)
	}

	for i := 0; i < len(backups); i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].ModifiedTime.After(backups[j].ModifiedTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	return backups, nil
}

func (b *SeaDexBackup) GetLatestBackup() (BackupFile, error) {
	backups, err := b.GetBackups()
	if err != nil {
		return BackupFile{}, err
	}
	if len(backups) == 0 {
		return BackupFile{}, fmt.Errorf("no backups found")
	}
	return backups[len(backups)-1], nil
}

func (b *SeaDexBackup) Download(file *BackupFile, destination string, overwrite bool) (string, error) {
	if destination == "" {
		var err error
		destination, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	info, err := os.Stat(destination)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s must be an existing directory", destination)
	}

	var key string
	if file == nil {
		latest, err := b.GetLatestBackup()
		if err != nil {
			return "", err
		}
		key = latest.Name
	} else {
		key = file.Name
	}

	outPath := filepath.Join(destination, key)
	if !overwrite {
		if _, err := os.Stat(outPath); err == nil {
			return "", fmt.Errorf("file already exists: %s", outPath)
		}
	}

	fileToken, err := b.getFileToken()
	if err != nil {
		return "", fmt.Errorf("getting file token: %w", err)
	}

	reqURL := fmt.Sprintf("%s/api/backups/%s?token=%s", b.baseURL, key, fileToken)
	resp, err := b.client.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpPath := outPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	succeeded := false
	defer func() {
		if !succeeded {
			os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	zr, err := zip.OpenReader(tmpPath)
	if err != nil {
		return "", newBadBackupFileError("%s failed integrity check: %v", outPath, err)
	}
	zr.Close()

	if err := os.Rename(tmpPath, outPath); err != nil {
		return "", err
	}

	succeeded = true
	return outPath, nil
}

var validBackupFilename = regexp.MustCompile(`^([a-z0-9_-]+\.zip)$`)

func (b *SeaDexBackup) Create(filename string) (BackupFile, error) {
	name := strings.TrimSuffix(filename, ".zip") + ".zip"
	name = strings.ToLower(time.Now().UTC().Format(name))

	if !validBackupFilename.MatchString(name) {
		return BackupFile{}, fmt.Errorf(
			"invalid filename %q: may only contain lowercase letters, numbers, hyphens, or underscores",
			name,
		)
	}

	payload, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequest(http.MethodPost, b.urlFor("/api/backups"), strings.NewReader(string(payload)))
	if err != nil {
		return BackupFile{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", b.adminToken)

	resp, err := b.client.Do(req)
	if err != nil {
		return BackupFile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return BackupFile{}, fmt.Errorf("create backup returned %d", resp.StatusCode)
	}

	backups, err := b.GetBackups()
	if err != nil {
		return BackupFile{}, err
	}
	for _, bf := range backups {
		if bf.Name == name {
			return bf, nil
		}
	}
	return BackupFile{}, fmt.Errorf("backup %q not found after creation", name)
}

func (b *SeaDexBackup) Delete(file BackupFile) error {
	req, err := http.NewRequest(http.MethodDelete, b.urlFor("/api/backups/"+file.Name), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", b.adminToken)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete backup returned %d", resp.StatusCode)
	}
	return nil
}
