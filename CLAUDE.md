# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

This workspace is my personal solo repo (`<github-user>/devops-group-b-solo`) for the Group B DevOps & Observability Research assignment. It follows the Plan Section 3 layout (`app/`, `.github/workflows/`, `k8s/`, `load-tests/`, `docs/`, `captures/`). The Docker build context is `app/`, and CI runs the Go and lint steps with `working-directory: app`. Personal `main` holds only Pillar 1; Pillar 2 goes on `p2` and Pillar 3 on `p3`.

The planning documents at the root are:

- `assignment-3-group-b-devops-observability-research.md`: the supervisor's brief. It is the source of truth for requirements, deliverables, and grading.
- `PROJECT_PLAN.md`: the team's execution plan (roles, stack decisions, schedule, checkpoint process, risks, Definition of Done).
- `CP1_USER_STORIES.md`: user stories and acceptance criteria for Checkpoint 1. It is the most precise spec for the app, image, and CI.

Read these documents for current dates, names, contracts, and decisions instead of relying on this file or on memory. They change often.

## Editing the documents

- The documents restate the same facts (dates, owners, app contract, CI rules, repo layout). When you change a fact in one, find and update every other place that states it.
- Edit the content directly. Don't add dated "Revision" lines or change logs; git history records what changed.
- The project plan is not part of the team repo. Content written for the team repo (docs, READMEs) must not link to it or depend on it.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <summary>`.

- **Types:** `feat`, `fix`, `test`, `refactor`, `build` (Dockerfile, Makefile, `go.mod`), `ci` (workflows), `docs`, `chore` (layout, config, captures).
- **Scopes** (optional) follow the repo layout: `app`, `docker`, `ci`, `k8s`, `monitoring`, `load-tests`, `docs`, `captures`.
- **Summary:** imperative mood, lowercase, no trailing period, 72 characters or fewer. Example: `feat(app): add /api/work with ms and fail params`.
- **Subject line only:** no body and no trailers.
- Keep commits small and focused: one logical change per commit. Progressive history is graded evidence of individual work (risk R6), so never squash a day's work into one commit.
- No `Co-Authored-By` or other AI attribution lines.
