package main

import (
	"flag"
	"fmt"
	"os"

	seadex "github.com/RAELIE1/seadexgo"
)

const backupUsage = `seadexgo backup — manage SeaDex backups (requires admin credentials)

Credentials are read from environment variables:
  SEADEX_EMAIL     admin e-mail address
  SEADEX_PASSWORD  admin password

Usage:
  seadexgo [--json] backup list
  seadexgo [--json] backup download [--name <filename>] [--dest <dir>] [--overwrite]
  seadexgo [--json] backup create  [--name <template>]
  seadexgo [--json] backup delete  --name <filename>

Subcommands:
  list      Print all available backups, newest last
  download  Download a backup (defaults to the latest) to --dest directory
  create    Trigger a new backup on the server
  delete    Delete a named backup from the server

Flags for download:
  --name       Exact backup filename to download (omit for latest)
  --dest       Destination directory (default: current working directory)
  --overwrite  Overwrite existing file if present

Flags for create:
  --name  Go time-format template for the filename (default: "backup-20060102-150405.zip")

Flags for delete:
  --name  Exact backup filename to delete (required)
`

func backupCreds() (email, password string) {
	email = os.Getenv("SEADEX_EMAIL")
	password = os.Getenv("SEADEX_PASSWORD")
	if email == "" || password == "" {
		die("backup: SEADEX_EMAIL and SEADEX_PASSWORD must be set")
	}
	return
}

func newBackupClient(baseURL string) *seadex.SeaDexBackup {
	email, password := backupCreds()
	var opts []func(*seadex.SeaDexBackup)
	if baseURL != "" {
		opts = append(opts, seadex.WithBackupBaseURL(baseURL))
	}
	client, err := seadex.NewSeaDexBackup(email, password, opts...)
	if err != nil {
		dieErr(err)
	}
	return client
}

func runBackup(args []string, baseURL string, p *printer) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, backupUsage)
		os.Exit(2)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "list":
		runBackupList(rest, baseURL, p)
	case "download":
		runBackupDownload(rest, baseURL, p)
	case "create":
		runBackupCreate(rest, baseURL, p)
	case "delete":
		runBackupDelete(rest, baseURL, p)
	default:
		fmt.Fprintf(os.Stderr, "seadexgo backup: unknown subcommand %q\n\n%s", sub, backupUsage)
		os.Exit(2)
	}
}

func runBackupList(args []string, baseURL string, p *printer) {
	fs := flag.NewFlagSet("backup list", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, backupUsage) }
	fs.Parse(args)

	client := newBackupClient(baseURL)
	defer client.Close()

	backups, err := client.GetBackups()
	if err != nil {
		dieErr(err)
	}
	p.printBackups(backups)
}

func runBackupDownload(args []string, baseURL string, p *printer) {
	fs := flag.NewFlagSet("backup download", flag.ExitOnError)
	nameFlag := fs.String("name", "", "exact backup filename (omit for latest)")
	destFlag := fs.String("dest", "", "destination directory (default: cwd)")
	overwriteFlag := fs.Bool("overwrite", false, "overwrite existing file")
	fs.Usage = func() { fmt.Fprint(os.Stderr, backupUsage) }
	fs.Parse(args)

	client := newBackupClient(baseURL)
	defer client.Close()

	var target *seadex.BackupFile
	if *nameFlag != "" {
		backups, err := client.GetBackups()
		if err != nil {
			dieErr(err)
		}
		for i, b := range backups {
			if b.Name == *nameFlag {
				target = &backups[i]
				break
			}
		}
		if target == nil {
			die("backup download: %q not found", *nameFlag)
		}
	}
	outPath, err := client.Download(target, *destFlag, *overwriteFlag)
	if err != nil {
		dieErr(err)
	}

	if p.jsonMode {
		p.printJSON(map[string]string{"path": outPath})
	} else {
		fmt.Printf("Downloaded: %s\n", outPath)
	}
}

func runBackupCreate(args []string, baseURL string, p *printer) {
	fs := flag.NewFlagSet("backup create", flag.ExitOnError)
	nameFlag := fs.String("name", "backup-20060102-150405.zip", "Go time-format template for the filename")
	fs.Usage = func() { fmt.Fprint(os.Stderr, backupUsage) }
	fs.Parse(args)

	client := newBackupClient(baseURL)
	defer client.Close()

	bf, err := client.Create(*nameFlag)
	if err != nil {
		dieErr(err)
	}

	if p.jsonMode {
		p.printJSON(bf)
	} else {
		fmt.Printf("Created:  %s\n", bf.Name)
		fmt.Printf("Size:     %s\n", humanBytes(bf.Size))
		fmt.Printf("Modified: %s\n", bf.ModifiedTime.Format("2006-01-02 15:04:05 UTC"))
	}
}

func runBackupDelete(args []string, baseURL string, p *printer) {
	fs := flag.NewFlagSet("backup delete", flag.ExitOnError)
	nameFlag := fs.String("name", "", "exact backup filename to delete (required)")
	fs.Usage = func() { fmt.Fprint(os.Stderr, backupUsage) }
	fs.Parse(args)

	if *nameFlag == "" {
		fmt.Fprint(os.Stderr, "backup delete: --name is required\n\n")
		fmt.Fprint(os.Stderr, backupUsage)
		os.Exit(2)
	}

	client := newBackupClient(baseURL)
	defer client.Close()

	backups, err := client.GetBackups()
	if err != nil {
		dieErr(err)
	}
	var target *seadex.BackupFile
	for i, b := range backups {
		if b.Name == *nameFlag {
			target = &backups[i]
			break
		}
	}
	if target == nil {
		die("backup delete: %q not found", *nameFlag)
	}

	if err := client.Delete(*target); err != nil {
		dieErr(err)
	}

	if p.jsonMode {
		p.printJSON(map[string]string{"deleted": *nameFlag})
	} else {
		fmt.Printf("Deleted: %s\n", *nameFlag)
	}
}
