---
description: Run the security review of a spec (design or code) and show the findings
argument-hint: <spec path or number> [design|code]
---

Run the security review for: $ARGUMENTS

1. Find the spec. If a number was given, use `specs/<number>-*.md` or `specs/done/<number>-*.md`.
2. Pick the mode: the second argument if given; otherwise `code` if `git diff origin/master` or `git status --short` shows changes, and `design` if they do not.
3. In code mode, if `git status --short` shows untracked files, run `git add -N .` so they show in the diff. Use the `security_coordinator` agent with the spec path, the mode and, in code mode, the diff command `git diff origin/master`.
4. Show its report as it is, including the "Pre-existing" and "Dropped" sections.
5. Do not change any file, do not commit and do not fix anything. Ask what to do with the findings.
