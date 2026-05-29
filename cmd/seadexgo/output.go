package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	seadex "github.com/RAELIE1/seadexgo"
)

type printer struct {
	jsonMode bool
	w        *tabwriter.Writer
}

func newPrinter(jsonMode bool) *printer {
	return &printer{
		jsonMode: jsonMode,
		w:        tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

func (p *printer) flush() { p.w.Flush() }

func (p *printer) printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		die("json encode: %v", err)
	}
}

func (p *printer) printEntries(entries []seadex.EntryRecord) {
	if p.jsonMode {
		dicts := make([]map[string]any, len(entries))
		for i, e := range entries {
			dicts[i] = e.ToDict()
		}
		p.printJSON(dicts)
		return
	}
	for i, e := range entries {
		if i > 0 {
			fmt.Fprintln(p.w)
		}
		p.renderEntry(e)
	}
	p.flush()
}

func (p *printer) printEntry(e seadex.EntryRecord) {
	if p.jsonMode {
		p.printJSON(e.ToDict())
		return
	}
	p.renderEntry(e)
	p.flush()
}

func (p *printer) renderEntry(e seadex.EntryRecord) {
	fmt.Fprintf(p.w, "Entry\t%s\n", e.ID)
	fmt.Fprintf(p.w, "  AniList ID\t%d\n", e.AnilistID)
	fmt.Fprintf(p.w, "  Collection\t%s (%s)\n", e.CollectionName, e.CollectionID)
	fmt.Fprintf(p.w, "  URL\t%s\n", e.URL)
	fmt.Fprintf(p.w, "  Incomplete\t%v\n", e.IsIncomplete)
	if e.Notes != "" {
		fmt.Fprintf(p.w, "  Notes\t%s\n", e.Notes)
	}
	if e.TheoreticalBest != nil {
		fmt.Fprintf(p.w, "  Theoretical Best\t%s\n", *e.TheoreticalBest)
	}
	if len(e.Comparisons) > 0 {
		fmt.Fprintf(p.w, "  Comparisons\t%s\n", strings.Join(e.Comparisons, ", "))
	}
	fmt.Fprintf(p.w, "  Total Size\t%s\n", humanBytes(e.Size))
	fmt.Fprintf(p.w, "  Updated\t%s\n", e.UpdatedAt.Format(time.RFC3339))
	fmt.Fprintf(p.w, "  Torrents\t%d\n", len(e.Torrents))
	for _, t := range e.Torrents {
		p.renderTorrent(t)
	}
}

func (p *printer) renderTorrent(t seadex.TorrentRecord) {
	best := ""
	if t.IsBest {
		best = " [BEST]"
	}
	dual := ""
	if t.IsDualAudio {
		dual = " [DA]"
	}
	fmt.Fprintf(p.w, "    Torrent\t%s%s%s\n", t.ID, best, dual)
	fmt.Fprintf(p.w, "      Group\t%s\n", t.ReleaseGroup)
	fmt.Fprintf(p.w, "      Tracker\t%s\n", string(t.Tracker))
	fmt.Fprintf(p.w, "      URL\t%s\n", t.URL)
	if t.Infohash != nil {
		fmt.Fprintf(p.w, "      Infohash\t%s\n", *t.Infohash)
	}
	if len(t.Tags) > 0 {
		tags := make([]string, len(t.Tags))
		for i, tag := range t.Tags {
			tags[i] = string(tag)
		}
		fmt.Fprintf(p.w, "      Tags\t%s\n", strings.Join(tags, ", "))
	}
	fmt.Fprintf(p.w, "      Size\t%s  (%d files)\n", humanBytes(t.Size), len(t.Files))
}

func (p *printer) printBackups(backups []seadex.BackupFile) {
	if p.jsonMode {
		p.printJSON(backups)
		return
	}
	fmt.Fprintf(p.w, "NAME\tSIZE\tMODIFIED\n")
	for _, b := range backups {
		fmt.Fprintf(p.w, "%s\t%s\t%s\n", b.Name, humanBytes(b.Size), b.ModifiedTime.Format(time.RFC3339))
	}
	p.flush()
}

func (p *printer) printFiles(files []seadex.File) {
	if p.jsonMode {
		p.printJSON(files)
		return
	}
	fmt.Fprintf(p.w, "FILENAME\tSIZE\n")
	for _, f := range files {
		fmt.Fprintf(p.w, "%s\t%s\n", f.Name, humanBytes(f.Size))
	}
	p.flush()
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n2 := n / unit; n2 >= unit; n2 /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "seadexgo: "+format+"\n", args...)
	os.Exit(1)
}

func dieErr(err error) {
	die("%v", err)
}
