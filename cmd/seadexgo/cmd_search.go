package main

import (
	"flag"
	"fmt"
	"os"

	seadex "github.com/RAELIE1/seadexgo"
)

const searchUsage = `seadexgo search — find SeaDex entries

Usage:
  seadexgo [--json] search --filter <pocketbase-filter>
  seadexgo [--json] search --filename <filename>
  seadexgo [--json] search --infohash <40-char hex>
  seadexgo [--json] search --all

Flags:
  --filter    Raw PocketBase filter expression (e.g. "isBest=true")
  --filename  Match entries that contain a torrent file with this name
  --infohash  Match entries by torrent infohash (40-char hex)
  --all       Dump every entry in SeaDex (slow — ~500 records per page)

The --all flag streams results one page at a time; output order is not guaranteed.
`

func runSearch(args []string, baseURL string, p *printer) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	filterFlag := fs.String("filter", "", "PocketBase filter expression")
	filenameFlag := fs.String("filename", "", "torrent filename to match")
	infohashFlag := fs.String("infohash", "", "40-char hex infohash")
	allFlag := fs.Bool("all", false, "dump all SeaDex entries")
	fs.Usage = func() { fmt.Fprint(os.Stderr, searchUsage) }
	fs.Parse(args)

	chosen := 0
	if *filterFlag != "" {
		chosen++
	}
	if *filenameFlag != "" {
		chosen++
	}
	if *infohashFlag != "" {
		chosen++
	}
	if *allFlag {
		chosen++
	}
	if chosen == 0 {
		fmt.Fprint(os.Stderr, searchUsage)
		os.Exit(2)
	}
	if chosen > 1 {
		die("search: --filter, --filename, --infohash, and --all are mutually exclusive")
	}

	client := newClient(baseURL)
	defer client.Close()

	var (
		entries []seadex.EntryRecord
		err     error
	)

	switch {
	case *filterFlag != "":
		entries, err = client.FromFilter(*filterFlag)
		if err != nil {
			dieErr(err)
		}

	case *filenameFlag != "":
		entries, err = client.FromFilename(*filenameFlag)
		if err != nil {
			dieErr(err)
		}

	case *infohashFlag != "":
		entries, err = client.FromInfohash(*infohashFlag)
		if err != nil {
			dieErr(err)
		}

	case *allFlag:
		entries, err = client.Iterator()
		if err != nil {
			dieErr(err)
		}
	}

	if len(entries) == 0 {
		if !p.jsonMode {
			fmt.Fprintln(os.Stderr, "no entries found")
		} else {
			fmt.Println("[]")
		}
		return
	}

	p.printEntries(entries)
}
