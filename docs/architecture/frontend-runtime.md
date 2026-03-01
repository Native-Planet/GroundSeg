# Frontend Runtime Architecture

This document defines ownership boundaries for the GroundSeg frontend runtime.

## Source Of Truth

- Authored frontend source lives in `ui/src/`.
- Runtime contracts and transport modules live in `ui/src/lib/runtime/`.
- Shared stores live in `ui/src/lib/stores/`.

Do not hand-edit generated bundles under `goseg/web/_app/immutable/`.
Those files are output from the UI build pipeline and are refreshed by building `ui/`.

## Package Boundaries

- `ui/src/lib/runtime/commands/`: domain command factories (`auth`, `system`, `urbit`, `support`, etc).
- `ui/src/lib/runtime/transport/`: websocket/SSE transport helpers and URL/parse utilities.
- `ui/src/lib/runtime/session/`: startup orchestration and runtime sequencing.
- `ui/src/lib/runtime/handlers/`: GroundSeg domain event routing from realtime transports.
- `ui/src/lib/runtime/config/`: immutable runtime mode config from injected env values.
- `ui/src/lib/stores/`: Svelte stores and route-facing adapters.
- `ui/src/lib/utils/`: non-store helpers (for example date calculations).

## Runtime Contract Rules

- UI routes/components should depend on domain command APIs and stores, not generated chunks.
- Transport adapters must expose explicit readiness/send semantics.
- Session/token persistence concerns must remain isolated from route components.

## Generated Artifact Policy

- Build artifacts are produced from `ui/` and published to `goseg/web/` for release packaging.
- Changes to generated assets are accepted only when produced by a corresponding `ui` build.
- Any runtime behavior change must be implemented in `ui/src/` first.

## Quality Gate

Before promoting or packaging frontend runtime changes:

1. `cd ui && npm ci`
2. `npm run test`
3. `npm run build`

If these checks fail, do not refresh `goseg/web/` artifacts.
