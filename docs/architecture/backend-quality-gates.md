# Backend Quality Gates

## Required Backend Checks

1. `cd goseg`
2. `go test ./...`
3. `go test -tags=integration ./broadcast ./handler ./routines ./ws`
   - Run integration targets only in environments with Docker/runtime deps.

## Runtime Boundary Expectations

1. Preserve `%w` error wrapping at handler/service boundaries.
2. Keep fetch-only APIs separate from mutating sync APIs (`Fetch*` vs `Sync*`).
3. For registration/version/upload/C2C path changes, include deterministic unit
   tests in changed packages.
4. Follow package governance docs for edge contracts:
   - `goseg/uploadsvc/GOVERNANCE.md`
   - `goseg/system/wifi/README.md`
   - `goseg/startram/GOVERNANCE.md`
   - `goseg/protocol/contracts/GOVERNANCE.md`
5. Use shared boundary helpers for dependency-injected handlers and masked error
   semantics (`goseg/errpolicy`).

## CI Runtime Contract Gate

`Runtime Contract Gate` (`.github/workflows/upload-contract-gate.yml`) enforces
path-aware test coverage when contract surfaces change.

Targeted checks are executed via
`.github/scripts/check-upload-contract.sh`, including protocol-family and
StarTram masked-error regression tests in `goseg/protocol/contracts`.
