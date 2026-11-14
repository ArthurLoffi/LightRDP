---
description: Implement a spec and open a pull request
argument-hint: <spec path or number, e.g. specs/0001-clipboard-sync.md or 0001>
---

Implement the spec: $ARGUMENTS

## 1. Read
1. Find the spec. If a number was given, use `specs/<number>-*.md`.
2. Read `AGENTS.md` and follow every rule in it.
3. Read the whole spec. Stop and ask if:
   - `Status` is not `Approved`.
   - "Open questions" still has unanswered items.
   - The spec needs a new dependency or a breaking change that the spec does not explicitly approve.
4. Read the code the spec touches before changing anything.

## 2. Branch
1. Run `git status`. If there are uncommitted changes, stop and ask.
2. Run `git fetch origin` and create a new branch from `origin/master`:
   - `feat/NNNN-<slug>` for features, `fix/NNNN-<slug>` for fixes.
   - Run `git branch --unset-upstream` so the branch does not track `master`.
3. Never commit to `master`.

## 3. Implement
1. Implement only what is in the spec's "Scope" and "Requirements". Write anything else you notice in the pull request, not in the code.
2. Follow the existing patterns of the codebase.
3. Do not add comments to the code.
4. Add tests for every requirement. Never weaken or special-case tests to make them pass.
5. Update:
   - `CHANGELOG.md`, under `[Unreleased]` with today's date.
   - `docs/app/` for the new feature or contract change.
   - `docs/errors/` for every new error in the spec's "Errors" section.
   - `README.md` if the getting-started instructions changed.

## 4. Verify
1. Run `gofmt -l .`, `go vet ./...` and `go test -race ./...`. All must pass.
2. Check every item in "Acceptance criteria". Mark the ones that are done with `[x]` in the spec. If one cannot be met, stop and ask.
3. Set the spec `Status` to `In review`.

## 5. Commit and pull request
1. Commit with a Conventional Commits message, for example `feat: clipboard sync (spec 0001)`. Do not add a co-author and do not mention yourself in the commit.
2. Push the branch: `git push -u origin <branch>`.
3. Open the pull request with `gh pr create --base master`, using `.github/PULL_REQUEST_TEMPLATE.md` as the body:
   - Goal and Context: from the spec, with a link to the spec file.
   - Changes: the main changes.
   - How to test: from the spec's "Test plan".
   - AI usage: the AI tool and model you are running as.
   - Checklist: mark only what was actually done.
4. Put the pull request URL in the spec's `Pull request` field, commit, and push.
5. Show the pull request URL, and list anything from the spec that was not done.
