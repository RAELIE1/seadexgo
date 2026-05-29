package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	seadex "github.com/RAELIE1/seadexgo"
)

const queryUsage = `seadexgo query — look up a single SeaDex entry

Usage:
  seadexgo [--json] query --id <anilist-int | pocketbase-string>
  seadexgo [--json] query --title <anime title>

Flags:
  --id    AniList integer ID (e.g. 21) or PocketBase record ID (e.g. "abc123xyz")
  --title Anime title to search via AniList (e.g. "Mushishi")
`

func runQuery(args []string, baseURL string, p *printer) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	idFlag := fs.String("id", "", "AniList int ID or PocketBase record ID")
	titleFlag := fs.String("title", "", "anime title (resolved via AniList)")
	fs.Usage = func() { fmt.Fprint(os.Stderr, queryUsage) }
	fs.Parse(args)

	if *idFlag == "" && *titleFlag == "" {
		fmt.Fprint(os.Stderr, queryUsage)
		os.Exit(2)
	}
	if *idFlag != "" && *titleFlag != "" {
		die("query: --id and --title are mutually exclusive")
	}

	client := newClient(baseURL)
	defer client.Close()

	var (
		entry seadex.EntryRecord
		err   error
	)

	switch {
	case *titleFlag != "":
		entry, err = client.FromTitle(*titleFlag)
		if err != nil {
			dieErr(err)
		}

	case *idFlag != "":
		if n, parseErr := strconv.Atoi(*idFlag); parseErr == nil {
			entry, err = client.FromID(n)
		} else {
			entry, err = client.FromID(*idFlag)
		}
		if err != nil {
			dieErr(err)
		}
	}

	p.printEntry(entry)
}

func newClient(baseURL string) *seadex.SeaDexEntry {
	if baseURL != "" {
		return seadex.NewSeaDexEntry(seadex.WithBaseURL(baseURL))
	}
	return seadex.NewSeaDexEntry()
}
