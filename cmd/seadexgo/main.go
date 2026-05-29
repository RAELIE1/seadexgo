package main

import (
	"flag"
	"fmt"
	"os"
)

const usageText = `seadexgo — SeaDex command-line client

Usage:
  seadexgo [global flags] <command> [flags] [args]

Commands:
  query    Look up a single entry by AniList ID, PocketBase ID, or title
  search   Find entries by custom filter, filename, or infohash
  backup   Manage and download SeaDex backups (requires admin credentials)
  torrent  Inspect and sanitize .torrent files

Global flags:
  --json          Output raw JSON instead of human-readable text
  --base-url URL  SeaDex base URL (default: https://releases.moe)

Run "seadexgo <command> --help" for command-specific help.
`

func main() {
	globalFlags := flag.NewFlagSet("seadexgo", flag.ContinueOnError)
	jsonMode := globalFlags.Bool("json", false, "output raw JSON")
	baseURL := globalFlags.String("base-url", "", "SeaDex base URL (overrides default)")
	globalFlags.Usage = func() { fmt.Fprint(os.Stderr, usageText) }

	if err := globalFlags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	args := globalFlags.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}

	p := newPrinter(*jsonMode)

	switch args[0] {
	case "query":
		runQuery(args[1:], *baseURL, p)
	case "search":
		runSearch(args[1:], *baseURL, p)
	case "backup":
		runBackup(args[1:], *baseURL, p)
	case "torrent":
		runTorrent(args[1:], p)
	default:
		fmt.Fprintf(os.Stderr, "seadexgo: unknown command %q\n\n%s", args[0], usageText)
		os.Exit(2)
	}
}
