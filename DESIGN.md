You are a backup-planning agent. Design a practical, privacy-focused backup system for a Linux/WSL-first operator.

Context:
- Host workflow runs from WSL command line on Windows.
- Source data includes:
  - WSL/Linux home and config data
  - Windows user directories (multiple users under C:\Users\...)
- Backup output should land on Windows filesystem paths (via /mnt/c/... from WSL), then be copied to external media and cloud.
- Exclusion strategy preference: deny-list (exclude patterns/paths), not strict allow-list.

Primary concerns to solve:
1) Monolith vs split outputs:
   - Avoid single giant artifact risk.
   - Recommend sizing/splitting strategy (hot vs cold, per-user, or per-data-class).
2) Encryption/privacy:
   - Strong client-side encryption.
   - Protect filenames/metadata where possible.
   - Minimize vendor lock-in.
3) Tooling:
   - Restic looks like the best fit for this use case, but assess alternatives (Borg, Duplicity, etc) for Windows/WSL compatibility and feature set.
4) Change visibility/reporting:
   - Must provide reports of added/changed/removed files between runs so deny-list can be tuned.
   - Must help detect unexpectedly large files, or directories with thousdands of small files, or just new directories.
5) Windows-specific restore fidelity:
   - Handle/assess ACLs, ADS, symlinks/junctions, and locked files (VSS/native capture considerations).
6) Reliability:
   - Integrity checks/scrubbing.
   - Restore testing cadence (file-level + full disaster scenario).
   - Corruption/blast-radius reduction strategy.
7) Security:
   - Ransomware threat model.
   - Secret/token separation from host.
   - Key/passphrase backup and recovery documentation.  I use KeePassXC for secrets management, so integration with that would be a plus.
8) Retention + lifecycle:
   - Different policy for hot vs cold sets.
   - Prune/forget/check cadence.
9) Cost:
   - Cheap cloud cold-storage options and hidden costs (egress/retrieval/API requests).
10) Governance:
   - Multi-user privacy/consent boundaries and access controls.
11) Environment:
   - Linux/WSL-first, but must handle Windows-specific features and constraints.
   - Solo operator with limited time and budget, so automation and simplicity are key.
   - Currently, I have an external hard drive.  In the future I will set up cloud storage, and later I will set up a local server for air-gapped backups.

Deliverables requested:
- A recommended target architecture (with rationale and tradeoffs).
- Repository layout proposal (how many repos, naming, hot/cold split).
- Concrete schedule (daily/weekly/monthly tasks).
- Example deny-list structure and maintenance loop.
- Reporting/alerting plan for changed/new/large files.
- Disaster-recovery runbook outline.
- Cost model template comparing at least 3 cloud options.
- “Start small then scale” implementation phases (Phase 1/2/3).
   - I want to start with just notes and repos, then expand to full data backup later.
- Clear “do this / avoid this” list.

Constraints:
- Commands and examples should be Linux/WSL-friendly.
- If containerization is suggested, use Podman conventions (not Docker); Podman runs in WSL2 Fedora 42.
- Keep recommendations pragmatic for a solo operator with budget constraints.