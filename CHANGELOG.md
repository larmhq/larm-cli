# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Validation errors now list the failing fields and their messages, in both text and `--output json` (`errors` key). Previously they rendered as a bare `validation_error:`.
- Rate-limited requests now fail right away with the 429 and its suggestion, instead of waiting on a `Retry-After` (or backoff) that would outlast the 30s request timeout and then failing with a timeout.
- Updated to larm-go v0.3.1 and Go 1.27.3.

## [0.4.0]

Status pages now render as a tree, with new commands for managing groups.

### Added

- `larm component-groups` subcommand (create/show/update/delete) for managing status page component groups.
- `--group <id>` flag on `larm components create` and `larm components update` to place a component inside (or move it out of) a group.

### Changed

- `larm status-pages show` now renders the components tree (groups with their components indented; ungrouped components at the top level), matching how the status page is displayed.

### Removed

- `larm components list`. Fetch the parent status page with `larm status-pages show <id>` to see its components.
