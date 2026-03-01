# Frontend Runtime Modules

`ui/src/lib/runtime/` is the source of truth for frontend runtime contracts.

## Package Boundaries

- `commands/`: domain command factories for websocket actions.
- `config/`: runtime mode/env parsing.
- `contracts/`: shared transport/service interface contracts.
- `handlers/`: app-specific event routing functions.
- `session/`: startup orchestration and lifecycle coordination.
- `transport/`: low-level websocket/SSE helpers.

Generated bundles under `goseg/web/_app/immutable/` are build artifacts, not hand-edited source.
