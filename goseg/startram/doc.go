// Package startram encapsulates StarTram registration, sync, and backup flows.
//
// Error-visibility policy for external API failures is enforced at this
// boundary via `wrapAPIConnectionError`: callers receive a stable masked message
// while the original cause remains on the error chain for observability.
package startram
