---
description: Create a new spec from specs/_TEMPLATE.md
argument-hint: <short description of the feature>
---

Create a new spec for: $ARGUMENTS

1. Read `AGENTS.md` and `specs/_TEMPLATE.md`.
2. Pick the next number: the highest `NNNN` in `specs/` and `specs/done/`, plus one, padded to 4 digits.
3. Create `specs/NNNN-<slug>.md`, where `<slug>` is a short kebab-case name for the feature.
4. Fill every section of the template from the description and from the current code. Read the code the feature will touch so the technical design, errors and breaking changes are accurate.
5. Set `Status` to `Draft`, `Created` to today's date, and `Author` to the git user name.
6. Write anything you could not decide under "Open questions". Do not make product decisions on your own.
7. In "Security review", keep in `Reviewers` only the reviewers that apply to what the spec touches:
   - `network`: the signaling server, WebSocket, TLS or certificates.
   - `concurrency`: state shared between goroutines, the hub, or several connections at once.
   - `errors`: new messages, new input, or new errors.
   - `logic`: the connection flow, host approval, the safety code or the chat.
8. Run the design review: follow `.claude/commands/security.md`, steps 2 to 4, with the spec path and mode `design`.
   - Write each finding of its report under "Open questions" as a question to the author, with the scenario and the suggested fix. Do not change the requirements yourself.
   - Set `Design review` to `Done (YYYY-MM-DD)` with the number of findings, for example `Done (2026-11-25), 2 findings in Open questions`.
   - Do not write the report's "Pre-existing" findings in the spec. Show them only in your answer.
9. Do not implement anything, do not create a branch, and do not commit.
10. Show the path of the new spec, the open questions and any pre-existing findings.
