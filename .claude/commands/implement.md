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

## 5. Security review
Nothing is committed or pushed before this step passes: the repository is public, and a vulnerability pushed once stays in the history.
1. Stage every change with `git add -A`, without committing, so new files show in the diff. Then use the `security_coordinator` agent with the spec path, mode `code` and the diff command `git diff origin/master`.
2. For each finding in "Findings" (`Origin: introduced`):
   1. Write a test that reproduces it and fails.
   2. Fix it, following the existing patterns and every rule in `AGENTS.md`.
   3. Run step 4 again. All checks must pass.
3. Stop and ask, without fixing, if a fix needs a breaking change, a new dependency, or a change to another feature or to the spec's scope.
4. Run the `security_coordinator` agent again, as a recheck, with only the reviewers that had findings and the findings they must check. Do at most 2 rechecks. If findings remain after the second, stop and ask.
5. Findings in "Pre-existing" are not fixed in this branch and are never written in the spec, the commit or the pull request. Keep them for the final answer to the author.
6. Set `Code review` in the spec's "Security review" to `Passed (YYYY-MM-DD)`, with the number of rechecks and of findings fixed, for example `Passed (2026-11-25), 1 recheck, 2 findings fixed`. Do not describe how to exploit them.
7. Set the spec `Status` to `In review`.

## 6. Commit and pull request
1. Commit with a Conventional Commits message, for example `feat: clipboard sync (spec 0001)`. Do not add a co-author and do not mention yourself in the commit.
2. Push the branch: `git push -u origin <branch>`.
3. Open the pull request with `gh pr create --base master`, using `.github/PULL_REQUEST_TEMPLATE.md` as the body:
   - Goal and Context: from the spec, with a link to the spec file.
   - Changes: the main changes, and one line with the security review result: the reviewers run and a short description of each finding fixed, with no exploit details.
   - How to test: from the spec's "Test plan".
   - AI usage: the AI tool and model you are running as.
   - Checklist: mark only what was actually done.
4. Put the pull request URL in the spec's `Pull request` field, commit, and push.
5. Show the pull request URL, list anything from the spec that was not done, and show the pre-existing security findings from step 5 to the author only.
