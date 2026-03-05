// Package uploadsvc parses websocket upload actions and dispatches validated
// commands to upload backends.
//
// Validation is strict by action contract: reset must never carry open-endpoint
// payload fields, and open-endpoint requires a complete endpoint/token payload.
// Canonical cross-package conformance assertions live in
// goseg/protocol/contracts/conformance. Canonical upload contract declarations
// live in goseg/protocol/contracts/governance/upload_manifest.go, with shared
// governance composition in goseg/protocol/contracts/governance/manifest.go and policy details in
// docs/architecture/contracts-governance.md.
package uploadsvc
