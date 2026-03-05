// Package startram coordinates StarTram registration, endpoint health checks,
// renewal, and backup/restore workflows.
//
// External API failures are normalized to the governed StarTram contract error
// identifiers so callers receive stable error semantics across transport modes.
package startram
