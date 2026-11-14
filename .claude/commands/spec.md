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
7. Do not implement anything, do not create a branch, and do not commit.
8. Show the path of the new spec and the open questions.
