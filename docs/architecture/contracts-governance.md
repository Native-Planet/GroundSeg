# Contracts Governance

This document is the canonical policy for protocol contract governance and
quality-gate ownership.

## Quality Gate

Before merging backend/runtime changes that touch contracts:

1. `cd goseg`
2. `go test ./protocol/contracts/... ./uploadsvc/...`

## Governance Entrypoints

1. Contract declarations:
   - `goseg/protocol/contracts/catalog/action/`
   - `goseg/protocol/contracts/catalog/startram/`
   - Backward-compat shim: `goseg/protocol/contracts/familycatalog/`
2. Governance validators and registry assembly:
   - `goseg/protocol/contracts/`
3. Conformance fixtures and contract checks:
   - `goseg/protocol/contracts/conformance/`
4. Consumer adapter checks against upload binding specs:
   - `goseg/uploadsvc/`

## Ownership

- `protocol/contracts` maintainers own declarations and governance validators.
- `uploadsvc` maintainers own consumer conformance at the upload boundary.
