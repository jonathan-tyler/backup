<!-- markdownlint-disable MD022 MD031 MD032 MD047 MD058 MD060 -->

# Backup System Design (WSL-first, privacy-focused)

Date: 2026-02-14
Operator profile: solo, Linux/WSL-first, limited time, budget target for cloud is $20/month.

## 1) Recommended target architecture

### Decision summary
- Primary tool: `restic`.
- Split repositories by data class and temperature (hot vs cold), not a single monolith.
- Local-first workflow: write backup repositories to Windows paths (`/mnt/c/...`) from WSL.
- Replicate those repositories to external drive now, cloud in Phase 3, air-gapped server later.
- Windows fidelity strategy: hybrid capture (WSL + native Windows snapshot step for locked files).

### Why this is the best fit
- `restic` is cross-platform, encrypted client-side, backend-agnostic, and easy to automate.
- Multiple repos reduce blast radius, corruption risk, and restore scope for an incident.
- Split hot/cold retention keeps daily operations cheap while preserving deep history for cold data.
- Backend independence reduces lock-in versus provider-specific backup products.

### Tradeoffs
- More repos means more credentials and more scheduled jobs.
- Filename privacy is encrypted at rest in restic repos, but path exposure can still occur in logs.
- Windows ACL/ADS fidelity is not perfect from WSL alone; native Windows capture is needed for
  high-fidelity sets.

## 2) Tooling assessment

### Recommended baseline: restic
- Strengths: strong encryption, snapshots, dedupe, many backends, simple CLI, good automation.
- WSL fit: excellent; can back up Linux and mounted Windows paths.
- Windows fit: works natively and supports volume snapshot usage for locked files.
- Reporting fit: supports snapshot diff and structured output usable for change reports.

### Alternatives (for context)
- Borg:
  - Excellent dedupe and security, strong Linux experience.
  - Weaker ergonomics for mixed Windows/WSL environments than restic.
- Duplicity:
  - Mature and cloud-capable.
  - More operational complexity for restores and reporting in this use case.
- Kopia:
  - Modern features and policy model.
  - Good option, but restic keeps the stack simpler and widely documented for WSL scripts.

## 3) Repository layout proposal

Use separate restic repositories by class and temperature.

### Repository names
- `r-hot-notes-repos`
- `r-hot-linux-home-core`
- `r-hot-win-users-active`
- `r-cold-media-archive`
- `r-cold-system-images` (optional later)

### Physical layout (Windows path, managed from WSL)
```text
/mnt/c/BackupRepos/
  r-hot-notes-repos/
  r-hot-linux-home-core/
  r-hot-win-users-active/
  r-cold-media-archive/
  r-cold-system-images/
```

### Split rationale
- `notes + repos` isolated for very fast, frequent backup and restore.
- Linux home core separate from Windows users to avoid giant combined snapshots.
- Cold archive isolated to avoid frequent scans of huge, low-change data.
- Optional per-user split if privacy/governance boundaries require it:
  `r-hot-win-user-alice`, `r-hot-win-user-bob`, etc.

## 4) Encryption, privacy, and secrets

### Encryption baseline
- Use restic defaults (AES-256 + Poly1305 in authenticated mode, with encrypted metadata).
- Do not store repository passwords in shell history or plain files under synced user folders.

### KeePassXC integration (pragmatic)
- Store each repo passphrase and cloud key in KeePassXC entries.
- Use dedicated attributes per repo (`RESTIC_PASSWORD`, `RESTIC_REPOSITORY`, backend keys).
- Use the `wsl-backup` CLI to resolve secrets at runtime from a local secure export step.

### Secret separation
- Keep passphrases and cloud API credentials separate from source data host where possible.
- Store offline printed recovery material for master secrets in sealed physical storage.

### Minimal lock-in approach
- Keep restic repos on filesystem/S3-compatible targets.
- Avoid provider-only backup formats.

## 5) Windows-specific restore fidelity

### Fidelity targets
- Linux/WSL files: high-fidelity for content, permissions, symlinks.
- Windows user files: content fidelity high; ACL/ADS and locked files require native step.

### Hybrid capture model
- WSL restic jobs for general files via `/mnt/c/Users/...`.
- For fidelity-sensitive/locked Windows data, run native Windows restic with snapshot option
  (VSS-backed), then store output into the same repo family naming convention.

### Expectation table
- ACLs: best effort from WSL; improved when captured natively.
- ADS: do not assume full preservation from WSL path traversal.
- Junctions/symlinks: verify per set; do not blindly recurse through junction loops.
- Locked files: require native snapshot capture.

## 6) Deny-list strategy and maintenance loop

Deny-list preference is preserved. Start broad, tune from change reports.

### Suggested layout
```text
backup/
  config/
    excludes/
      common.exclude
      linux-home.exclude
      win-users.exclude
      cold-media.exclude
```

### Example `common.exclude`
```text
# Caches and temp
**/.cache/**
**/Cache/**
**/tmp/**
**/*.tmp

# Package/build artifacts
**/node_modules/**
**/dist/**
**/build/**

# VCS internals
**/.git/objects/**
**/.git/lfs/objects/**

# OS noise
**/Thumbs.db
**/Desktop.ini
```

### Example `win-users.exclude`
```text
# Large regenerable app data
/mnt/c/Users/*/AppData/Local/Temp/**
/mnt/c/Users/*/AppData/Local/Packages/**/LocalCache/**

# Browser caches (keep profiles only if needed)
/mnt/c/Users/*/AppData/Local/*/User Data/Default/Cache/**
```

### Maintenance loop
1. Run backup and produce change report.
2. Review new directories, large files, and high small-file-count directories.
3. Classify each finding: keep, move to cold set, or exclude.
4. Update deny-list files and commit changes to this repo.
5. Re-run dry check on next cycle.

## 7) Reporting and alerting plan

### Required output per run
- Added/changed/removed counts by repo.
- Top N largest new/changed files.
- New directories detected.
- Directories with very high file counts (small-file churn hotspots).

### Practical approach
- Use restic snapshot comparison between latest and previous snapshot.
- Emit machine-readable output (`--json`) and summarize in a daily text report.
- Keep rolling reports under `/mnt/c/BackupReports/` and mirror to external drive.

### Alert thresholds (initial)
- New file > 2 GiB in hot sets.
- Directory adds > 5,000 files in one day.
- Any brand-new top-level directory under protected roots.

### Delivery
- Phase 1: local report files + terminal summary.
- Phase 2: optional email/webhook notification for threshold breaches.

## 8) Reliability, integrity, and blast-radius reduction

### Reliability controls
- Independent repos by class reduce corruption impact.
- Schedule periodic `restic check` (metadata and pack integrity).
- Keep at least one offline copy on external media disconnected after sync.

### Cadence
- Daily: backup runs + report generation.
- Weekly: prune/forget + quick spot restore test.
- Monthly: full `check` on one rotating repo + larger restore drill.
- Quarterly: disaster simulation from bare target to working state.

### Restore testing policy
- File-level test every week from each hot repo.
- Full scenario test quarterly for one repo, rotating until all covered.

## 9) Retention and lifecycle policies

### Hot repos (notes/repos, active home, active user data)
- Keep `30` daily, `12` weekly, `12` monthly.
- Prune weekly.

### Cold repos (media/archive)
- Keep `14` daily, `8` weekly, `24` monthly, `5` yearly.
- Prune monthly.

### Lifecycle path
- Hot data that cools down over time can be migrated into cold repo.
- Document move decisions in a simple `CHANGELOG.md` for auditability.

## 10) Concrete operating schedule

### Daily
- Run hot backups.
- Generate and review summary report (focus on unexpected adds/large files).

### Weekly
- Run cold backups.
- Run retention prune/forget jobs.
- Restore 3-5 random files from hot repos.

### Monthly
- Run integrity checks (`restic check`) on all hot repos and one cold repo.
- Restore one whole directory tree from backup to test location.

### Quarterly
- Execute disaster recovery drill from documented runbook.
- Review and rotate credentials/tokens where required.

## 11) Ransomware and security posture

### Threat model assumptions
- Primary risk: host compromise or encrypt/delete attack on online backup paths.

### Controls
- Maintain immutable/offline copy on disconnected external disk after sync.
- Use separate credentials for local replication and cloud backend.
- Enforce least privilege on cloud keys (backup-only where possible).
- Keep repo passwords outside normal user profile sync paths.

### Recovery materials
- Keep backup key/passphrase recovery document offline and tested.
- Include exact recovery commands and repository map in runbook.

## 12) Governance for multi-user data

### Privacy boundaries
- Prefer per-user repos when users have separate consent boundaries.
- If sharing a repo, enforce path-level policy and strict access to credentials.

### Access control
- Separate credentials by repo.
- Log who can restore which repo.

## 13) Disaster recovery runbook (outline)

1. Incident declaration and scope (what failed, what must be restored first).
2. Verify trusted environment before restore (clean host or fresh system).
3. Retrieve credentials from KeePassXC offline process.
4. Inventory available repos and snapshots.
5. Restore priority order:
   - `r-hot-notes-repos`
   - `r-hot-linux-home-core`
   - `r-hot-win-users-active`
   - Cold repos as needed
6. Validate restored data integrity and app usability.
7. Post-incident review and deny-list/report threshold updates.

## 14) Cost model template (3 cloud options)

Use this template with your measured monthly data growth.

### Inputs
- `S_total_gb`: total stored GB
- `S_new_gb`: new GB per month
- `R_restore_gb`: restore GB per month
- `N_put`, `N_get`: API request counts

### Formula
- Monthly cost = storage + retrieval/egress + API requests + minimum duration penalties

### Compare options (example rows)
| Provider | Storage $/GB-mo | Egress/Retrieval | API cost | Hidden costs to watch |
|---|---:|---:|---:|---|
| Backblaze B2 | input | input | input | Egress above free tier, lifecycle ops |
| Cloudflare R2 | input | input | input | Class B operations, provider compatibility |
| AWS S3 Glacier DA | input | input | input | Retrieval delay, request fees, min duration |

### Budget guardrail
- Keep cloud in Phase 3 only.
- Gate rollout if projected monthly > $20 using realistic restore assumptions.

## 15) Start small, then scale (implementation phases)

### Phase 1 (now): notes + repos only
Scope:
- Implement `r-hot-notes-repos` only.
- Daily backups, weekly restore sample, daily report summary.

Exit criteria:
- 30 consecutive days successful backups.
- At least 4 successful weekly restore samples.
- Deny-list stabilized (no recurring false positives).

### Phase 2: broader local data + reporting hardening
Scope:
- Add `r-hot-linux-home-core` and `r-hot-win-users-active`.
- Add threshold alerts and monthly integrity checks.

Exit criteria:
- 60 days stable runs across all hot repos.
- Quarterly drill completed once.

### Phase 3: cloud + future air-gapped server
Scope:
- Mirror repos to cloud under $20 cap.
- Add later replication to local air-gapped server.

Exit criteria:
- Verified restore from cloud and from air-gapped copy.
- Documented RPO/RTO measurements.

## 16) Do this / avoid this

### Do this
- Use multiple restic repos by class and hot/cold split.
- Keep reports and deny-list updates as first-class operational tasks.
- Test restores regularly, not only backups.
- Keep one offline copy disconnected between syncs.
- Keep recovery documentation offline and current.

### Avoid this
- Avoid one giant monolithic repo for all data.
- Avoid storing plaintext backup passwords in shell scripts or synced folders.
- Avoid assuming WSL-only capture preserves all Windows metadata.
- Avoid enabling cloud before local restore/testing discipline is stable.
- Avoid silent failures; every run should produce an explicit report.

## 17) Minimal WSL-friendly command skeleton

These are examples for a Phase 1 baseline and should be represented as
`wsl-backup` subcommands.

```bash
# Set repository and password source (example only)
export RESTIC_REPOSITORY=/mnt/c/BackupRepos/r-hot-notes-repos
export RESTIC_PASSWORD_COMMAND='keepassxc-cli show Backup/restic-notes -a password'

# Backup notes/repos with deny-list
restic backup \
  "$HOME/notes" "$HOME/repos" \
  --exclude-file ./config/excludes/common.exclude \
  --tag hot --tag notes-repos

# Keep retention policy for hot set
restic forget --prune --keep-daily 30 --keep-weekly 12 --keep-monthly 12

# Compare latest snapshot pair for change reporting
restic diff latest latest~1
```

If native Windows snapshot capture is needed, run a Windows-side restic command for locked
files and keep the same repo naming pattern.