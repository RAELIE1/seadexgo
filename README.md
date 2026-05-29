<br/>
<p align="center">
  <a href="https://github.com/RAELIE1/seadexgo">
    <img src="https://raw.githubusercontent.com/Ravencentric/seadex/refs/heads/main/docs/assets/logo.png" alt="Logo" width="200">
  </a>
  <p align="center">
    Go client for the SeaDex API.
  </p>
</p>

# SeaDexGo

SeaDexGo is a Go client for the SeaDex API. It provides access to SeaDex entry records, torrent metadata, and backup downloads in a native Go package.

## Overview

This package exposes two primary clients:

- `SeaDexEntry` for querying SeaDex entries by AniList ID, SeaDex ID, filename, infohash, or custom filters.
- `SeaDexBackup` for authenticating with the SeaDex backup service, listing backup files, and downloading archives with integrity validation.

The package is designed for use in Go applications and supports custom HTTP clients for testing or network configuration.

## Installation

Install the published module from GitHub:

```bash
go get github.com/RAELIE1/seadexgo@latest
```

If the module is used locally, ensure the module path in `go.mod` matches the desired import path.

## Usage

```go
package main

import (
    "fmt"
    "log"

    seadex "github.com/RAELIE1/seadexgo"
)

func main() {
    client := seadex.NewSeaDexEntry()

    entry, err := client.FromID(165790)
    if err != nil {
        log.Fatalf("failed to load entry: %v", err)
    }

    fmt.Printf("Title ID: %d\n", entry.AnilistID)
    fmt.Printf("Total size: %d\n", entry.Size)
}
```

### Backup client

```go
backup, err := seadex.NewSeaDexBackup("email@example.com", "password")
if err != nil {
    log.Fatalf("backup auth failed: %v", err)
}

files, err := backup.GetBackups()
if err != nil {
    log.Fatalf("failed to list backups: %v", err)
}

fmt.Printf("found %d backups\n", len(files))
```

## CLI

A command-line client is included under `cmd/seadexgo`.

### Install

```bash
# from source
make install

# or directly
go install github.com/RAELIE1/seadexgo/cmd/seadexgo@latest
```

### Commands

| Command | Description |
|---------|-------------|
| `query` | Look up a single entry by AniList ID, PocketBase record ID, or anime title |
| `search` | Find entries by PocketBase filter expression, filename, infohash, or dump all |
| `backup` | List, download, create, or delete SeaDex backups (requires admin credentials) |
| `torrent` | Inspect file lists and sanitize private `.torrent` metadata |

All commands accept a `--json` flag to emit machine-readable JSON instead of the default tabular output. A `--base-url` flag overrides the default `https://releases.moe` endpoint.

### Examples

```bash
# Look up by AniList ID
seadexgo query --id 21

# Look up by anime title (resolved via AniList)
seadexgo query --title "Mushishi"

# Look up by PocketBase record ID
seadexgo query --id abc123xyz456

# Find entries matching a custom PocketBase filter
seadexgo search --filter "isBest=true"

# Find entries by torrent filename
seadexgo search --filename "[SubsPlease] Dungeon Meshi - 01 (1080p).mkv"

# Find entries by infohash
seadexgo search --infohash a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2

# Dump every entry as JSON (pipe-friendly)
seadexgo --json search --all | jq '.[].anilist_id'

# List all available backups
export SEADEX_EMAIL=admin@example.com
export SEADEX_PASSWORD=secret
seadexgo backup list

# Download the latest backup to /tmp
seadexgo backup download --dest /tmp

# Download a specific backup, overwriting if it exists
seadexgo backup download --name backup-20240101-120000.zip --dest /tmp --overwrite

# Create a new backup on the server
seadexgo backup create

# Delete a backup
seadexgo backup delete --name backup-20240101-120000.zip

# Print the files contained in a torrent
seadexgo torrent filelist release.torrent

# Remove private tracker metadata and write a sanitized copy
seadexgo torrent sanitize private.torrent --dst public.torrent
```

### Backup credentials

`backup` subcommands require admin credentials supplied via environment variables:

```
SEADEX_EMAIL      admin e-mail address
SEADEX_PASSWORD   admin password
```

## Testing

Run the test suite with:

```bash
go test ./...
```

## License

Distributed under the MIT License. See `LICENSE` for details.
