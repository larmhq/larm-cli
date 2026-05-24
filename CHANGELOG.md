# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0]

Status pages now render as a tree, with new commands for managing groups.

### Added

- `larm component-groups` subcommand (create/show/update/delete) for managing status page component groups.
- `--group <id>` flag on `larm components create` and `larm components update` to place a component inside (or move it out of) a group.

### Changed

- `larm status-pages show` now renders the components tree (groups with their components indented; ungrouped components at the top level), matching how the status page is displayed.

### Removed

- `larm components list`. Fetch the parent status page with `larm status-pages show <id>` to see its components.
