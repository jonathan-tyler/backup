# AGENTS.md

## Project Intent

- This project is a thin, predictable wrapper around `restic`.
- The goal is a familiar CLI interface and gap-filling behavior, not a full backup engine rewrite.

## CLI Option Policy

- Do not add conditional CLI flags without asking the user first.
- For optional or environment-specific behavior, prefer config-file options over new CLI flags.
