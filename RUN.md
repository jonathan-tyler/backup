# Backup Run Guide

Current scope is `notes + repos` into `r-hot-notes-repos`.

Entrypoint model: use the single `wsl-backup` executable with subcommands
(`backup`, `report`, `forget-prune`, `restore-smoke`) rather than separate
per-action scripts.

## 1) One-time setup

1. Copy the environment template:

   ```bash
   cp config/backup.env.example config/backup.env
   ```

2. Edit `config/backup.env` for your local paths and KeePassXC entry name.
3. Edit `config/includes/notes-repos.include` with one source path per line.
4. Ensure dependencies are installed:

   ```bash
   command -v restic
   command -v keepassxc-cli
   ```

5. Install the CLI with `pipx` from this repo:

   ```bash
   pipx install /home/him/my/sys/linux/backup
   ```

## 2) Daily operations

Run backup:

```bash
wsl-backup backup
```

Generate report:

```bash
wsl-backup report
```

## 3) Weekly operations

Apply retention:

```bash
wsl-backup forget-prune
```

Run restore smoke test:

```bash
wsl-backup restore-smoke
```

Use a custom env file by adding a second positional argument:

```bash
wsl-backup backup /path/to/backup.env
```

## 4) Optional cron examples (WSL)

```bash
# Daily backup at 21:00
0 21 * * * cd /home/him/my/sys/linux/backup && wsl-backup backup

# Daily report at 21:20
20 21 * * * cd /home/him/my/sys/linux/backup && wsl-backup report

# Weekly prune + restore test Sunday 10:00
0 10 * * 0 cd /home/him/my/sys/linux/backup && \
   wsl-backup forget-prune && wsl-backup restore-smoke
```

## 5) Output locations

- Backup repo: `RESTIC_REPOSITORY` in `config/backup.env`
- Reports and logs: `REPORT_DIR` in `config/backup.env`

## 6) Test commands

Fast unit tests:

```bash
pytest -q tests -k "not integration"
```

Integration tests (requires `restic` in container):

```bash
pytest -q tests -k integration
```

Full suite:

```bash
pytest -q
```

## 7) Manual test scenario (similar to integration flow)

This creates throwaway dirs and files, runs two backups, generates a report,
and runs a restore smoke test.

Note: the backup engine is `restic` (often mistyped as "rustic").

### 7.1 Create test workspace and env file

```bash
TEST_ROOT="$(mktemp -d /tmp/wsl-backup-manual-XXXXXX)"
mkdir -p "$TEST_ROOT"/{notes,repos,reports,restic-repo}

cat > "$TEST_ROOT/manual.env" <<EOF
RESTIC_REPOSITORY=$TEST_ROOT/restic-repo
RESTIC_PASSWORD_COMMAND=printf manual-password
SOURCE_INCLUDE_FILE=$TEST_ROOT/sources.include
EXCLUDE_COMMON_FILE=$PWD/config/excludes/common.exclude
EXCLUDE_SET_FILE=$PWD/config/excludes/notes-repos.exclude
REPORT_DIR=$TEST_ROOT/reports
LARGE_FILE_THRESHOLD=1
HOTSPOT_THRESHOLD=1
KEEP_DAILY=30
KEEP_WEEKLY=12
KEEP_MONTHLY=12
EOF

cat > "$TEST_ROOT/sources.include" <<EOF
$TEST_ROOT/notes
$TEST_ROOT/repos
EOF

echo "Manual test root: $TEST_ROOT"
```

### 7.2 Seed a handful of files

```bash
printf 'first note\n' > "$TEST_ROOT/notes/note-1.txt"
printf 'repo file one\n' > "$TEST_ROOT/repos/repo-file-1.txt"
mkdir -p "$TEST_ROOT/notes/subdir-a"
printf 'nested note\n' > "$TEST_ROOT/notes/subdir-a/note-2.txt"
printf 'small file\n' > "$TEST_ROOT/notes/subdir-a/note-3.txt"
```

### 7.3 Run first backup

```bash
python -m scripts.cli backup "$TEST_ROOT/manual.env"
```

Expected output highlights:

- First run only: `Initializing repository: .../restic-repo`
- `Starting backup at ...`
- `Repo: .../restic-repo`
- JSON lines from `restic backup --json` (status/progress + summary)
- `Backup completed at ...`
- `Log written: .../reports/backup-<timestamp>.log`

### 7.4 Change files and run second backup

```bash
printf 'second note\n' > "$TEST_ROOT/notes/note-4.txt"
printf 'updated content\n' >> "$TEST_ROOT/repos/repo-file-1.txt"
rm "$TEST_ROOT/notes/note-1.txt"

python -m scripts.cli backup "$TEST_ROOT/manual.env"
```

Expected behavior:

- No repository initialization message this time.
- Another backup log file is created under `reports/`.
- restic JSON summary should indicate changed/new/removed paths between snapshots.

### 7.5 Run report and inspect generated files

```bash
python -m scripts.cli report "$TEST_ROOT/manual.env"
ls -1 "$TEST_ROOT/reports"

LATEST_REPORT="$(ls -1t "$TEST_ROOT/reports"/report-*.txt | head -n1)"
LATEST_DIFF="$(ls -1t "$TEST_ROOT/reports"/diff-*.txt | head -n1)"

echo "Report file: $LATEST_REPORT"
echo "Diff file:   $LATEST_DIFF"
sed -n '1,200p' "$LATEST_REPORT"
```

Expected report content:

- Header block:
  - `Backup report`
  - `Generated: ...`
  - `Repository: .../restic-repo`
- Snapshot block (after at least 2 backups):
  - `Previous snapshot: <id>`
  - `Current snapshot:  <id>`
- Changes summary:
  - `- added: ...`
  - `- changed: ...`
  - `- removed: ...`
- `New directories` section from restic diff lines ending with `/`
- `Diff log: .../diff-<timestamp>.txt`
- `Large files over 1` section (will list most files because threshold is low)
- `Small-file hotspots over 1 files` section

Expected diff file content (`diff-*.txt`):

- Raw `restic diff` output, usually lines prefixed with:
  - `+` for additions
  - `-` for removals
  - `M` / `U` / `T` for metadata/content/type changes

### 7.6 Run restore smoke test

```bash
python -m scripts.cli restore-smoke "$TEST_ROOT/manual.env"
```

Expected output highlights:

- `Running restore smoke test at ...`
- `Temporary restore path: /tmp/.../restore`
- restic restore progress output
- `Restore smoke test complete. Restored files: <number>`

Notes:

- `restore-smoke` restores paths listed in `SOURCE_INCLUDE_FILE`.
- Temporary restore files are deleted automatically at the end of the command.

### 7.7 Optional cleanup

```bash
rm -rf "$TEST_ROOT"
```
