// Package uploadsvc defines the upload command contract shared by websocket
// transport adapters and service executors.
//
// Governance for this boundary is encoded directly in code:
//   - `Action` is the typed protocol enum used by adapters and tests.
//   - `ParseAction` is the single parser/validator for incoming action strings.
//   - `SupportedActions` and command dispatch must remain aligned with the
//     internal action binding table.
//   - `DescribeAction` must be deterministic for supported and unsupported
//     actions.
package uploadsvc
