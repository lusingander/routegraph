# routegraph

routegraph is a static analyzer for Go web router definitions.

It currently focuses on [Echo](https://echo.labstack.com/) and extracts route
registrations such as `Group`, `GET`, `POST`, `Any`, and `Add` into a flat
endpoint list. Nested groups, simple string constants, limited function splits,
struct fields, chained group calls, and simple route tables are supported.

## Install

```sh
go install github.com/lusingander/routegraph/cmd/routegraph@latest
```

## CLI

Run `routegraph` with a target directory or package pattern:

```sh
routegraph ./...
routegraph ./internal/server
```

Example output:

```text
GET   /api/v1/users        listUsers   internal/routes/user.go:24
POST  /api/v1/users        createUser  internal/routes/user.go:25
GET   /api/v1/admin/stats  stats       internal/routes/admin.go:18
```

Use `--json` for machine-readable output:

```sh
routegraph --json ./...
```

Unknown dynamic paths are kept instead of being dropped:

```text
GET  <unknown>/users  listUsers  internal/routes/user.go:31
```

## Library

routegraph can also be used as a Go library via `routegraph.Analyze`.
The library API is intentionally small for now.

## Development

This project is implemented with the help of Codex.
