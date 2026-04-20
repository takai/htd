# htd

Headless To-Do (htd) is a headless task management system that enables both AI agents and CLI users to operate a structured workflow.

## Directory Structure

- Follow the standard Go project layout.

## Documentation

- @docs/prd.md  Product Requirements Document
- @docs/cli.md  CLI Specification
- @docs/datamodel.md Data Model Specification

## Build Commands

- `mise run build` — build the binary.
- `mise run test` — run the full test suit.
- `mise run lint` — run lint.

## Development Guidelines

- Use the red/green/refactor TDD cycle.
- Use `git ai-commit` to create commits.
  - Always include a short, clear summary in English using the `--context` option.

## Pull Request Workflow

1. Always work on a feature branch.
2. Update the documentation alongside the code.
3. Close the original issue via the PR.
