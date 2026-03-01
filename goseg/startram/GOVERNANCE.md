# StarTram Error Governance

This package owns outbound StarTram error-shaping at the package boundary.
The root `README.md` should link here; package docs are the contract source of truth.

## Error contract
- External API transport failures must be returned through `wrapAPIConnectionError`.
- Outward-facing message is stable and masked by
  `apiConnectionErrorMessage` (`"Unable to connect to API server"`).
- Cause chains must be preserved so internal observability is retained
  via `errors.Is`/`errors.Unwrap`.

## Compatibility invariants
- Do not expose upstream transport details (URL/query/body) through user-facing
  wrapped messages.
- Preserve the original error as the wrapped cause.

## Test anchors
Boundary assertions should remain in:
- `goseg/startram/errors_test.go`
  - `TestWrapAPIConnectionErrorRedactsUpstreamDetailsAndPreservesCause`
  - `TestWrapAPIConnectionErrorRetainsStableMessageWithoutPubkey`

## Usage expectations
- Use `wrapAPIConnectionError` at all package boundaries that return external
  API connection failures.
- Treat this message as a backward-compatibility surface for callers and UI text.
