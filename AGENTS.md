# AGENTS.md

## Skill Discovery (Check First)

Before implementing changes, check if there are applicable skills for reusable guidance and templates:

- `/home/developer/.agents/skills/`

## Project Intent

- This project is a thin, predictable wrapper around `restic`.
- The goal is a familiar CLI interface and gap-filling behavior, not a full backup engine rewrite.

## CLI Option Policy

- Keep the CLI surface minimal and predictable.
- Do not add conditional CLI flags without asking the user first.
- For optional or environment-specific behavior, prefer config-file options over new CLI flags.
