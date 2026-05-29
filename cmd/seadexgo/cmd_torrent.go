package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	seadex "github.com/RAELIE1/seadexgo"
)

func runTorrent(args []string, p *printer) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, torrentUsageText)
		os.Exit(2)
	}

	switch args[0] {
	case "filelist":
		runTorrentFilelist(args[1:], p)
	case "sanitize":
		runTorrentSanitize(args[1:], p)
	default:
		fmt.Fprintf(os.Stderr, "seadexgo torrent: unknown command %q\n\n%s", args[0], torrentUsageText)
		os.Exit(2)
	}
}

const torrentUsageText = `seadexgo torrent — Torrent file helpers

Usage:
  seadexgo [global flags] torrent <command> [flags] [args]

Commands:
  filelist   Print files contained in a .torrent
  sanitize   Remove private tracker metadata from a .torrent
`

func runTorrentFilelist(args []string, p *printer) {
	fs := flag.NewFlagSet("torrent filelist", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: seadexgo [--json] torrent filelist <path>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}

	torrent, err := seadex.NewSeaDexTorrent(fs.Arg(0))
	if err != nil {
		dieErr(err)
	}
	files, err := torrent.FileList()
	if err != nil {
		dieErr(err)
	}
	p.printFiles(files)
}

func runTorrentSanitize(args []string, p *printer) {
	fs := flag.NewFlagSet("torrent sanitize", flag.ExitOnError)
	dst := fs.String("dst", "", "destination torrent path")
	overwrite := fs.Bool("overwrite", false, "overwrite destination")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: seadexgo torrent sanitize <src> [--dst path] [--overwrite]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}

	torrent, err := seadex.NewSeaDexTorrent(fs.Arg(0))
	if err != nil {
		dieErr(err)
	}
	out, err := torrent.Sanitize(*dst, *overwrite)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			die("destination file already exists and overwrite is false")
		}
		dieErr(err)
	}
	if p.jsonMode {
		p.printJSON(map[string]any{"path": out})
		return
	}
	fmt.Fprintf(os.Stdout, "Saved sanitized torrent to %s\n", out)
}
