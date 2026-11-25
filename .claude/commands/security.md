---
description: Run the security review of a spec (design or code) and show the findings
argument-hint: <spec path or number> [design|code]
---

Run the security review for: $ARGUMENTS

This file is also the security review procedure that `/spec` and `/implement` follow. When one of them calls it, it gives the mode and, for a recheck, the reviewers and the findings to check, and it handles the report itself.

## 1. Spec and mode
1. Find the spec. If a number was given, use `specs/<number>-*.md` or `specs/done/<number>-*.md`.
2. Pick the mode: the second argument if given; otherwise `code` if `git diff origin/master` or `git status --short` shows changes, and `design` if they do not.
3. In code mode, if `git status --short` shows untracked files, run `git add -N .` so they show in the diff. The diff command is `git diff origin/master`.

## 2. Reviewers
1. Read `Reviewers` in the spec's `## Security review` section. Each name is the agent `security_<name>`:
   - `network`: `security_network` (WebSocket and TLS, client and server)
   - `concurrency`: `security_concurrency` (race conditions)
   - `errors`: `security_errors` (exceptional conditions and error messages)
   - `logic`: `security_logic` (connection flow and business logic)
2. If the section is missing or `Reviewers` is empty, use all four.
3. Skip names that are not in this list, and note them for the triage.
4. For a recheck, run only the reviewers the caller names.

## 3. Run the reviewers
Start all of them at once, in parallel, each as its own agent. Give each one:
- The mode, the spec path and, in code mode, the diff command.
- For a recheck, the findings it must check.

## 4. Triage
Use the `security_triage` agent with the spec path, the mode, the diff command in code mode, any note from step 2, and the raw output of every reviewer with its name.

## 5. Report
When run as `/security`:
1. Show the triage report as it is, including the "Pre-existing" and "Dropped" sections.
2. Do not change any file, do not commit and do not fix anything. Ask what to do with the findings.
