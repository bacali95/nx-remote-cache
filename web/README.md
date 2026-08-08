# nx-remote-cache admin UI

React + TypeScript + Vite + shadcn/ui frontend for the admin API in
`internal/adminapi`. Built output (`dist/`) is embedded into the Go binary
via `go:embed` (see `embed.go`) and served under `/admin/`.

```bash
npm ci
npm run build   # produces dist/, required before `go build` on the main module
npm run dev     # dev server with API proxy to :3000, see vite.config.ts
```

See the [repo root README](../README.md) for the full project — running
the whole stack, configuration, auth model.
