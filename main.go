package main

import (
	"os"

	backup "wsl-backup-cli/src"
)

func main() {
	os.Exit(backup.RunCLI(os.Args[1:], os.Stdout, os.Stderr, backup.SystemExecutor{}))
}
