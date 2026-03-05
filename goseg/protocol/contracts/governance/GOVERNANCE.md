# Contract Governance Manifest

This package is the canonical machine-readable source for:

- Protocol action declarations (namespace/action/id/owner/payload rules).
- StarTram contract declarations (id/name/description/message/owner).

Downstream catalogs and conformance tests should derive contract details from
`manifest.go` rather than duplicating literals in multiple places.

Validate with:

```bash
cd goseg
go test ./protocol/contracts/governance ./protocol/contracts/... ./protocol/actions/...
```
