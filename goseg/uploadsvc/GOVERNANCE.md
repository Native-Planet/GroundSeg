# Upload Service Governance

This package owns the upload command contract boundary for websocket-triggered uploads.
The root `README.md` should reference this file, but package-level policy is authoritative.

## Contract surface
- `Action` is the transport contract enum.
  - Current supported actions: `open-endpoint`, `reset`.
- `SupportedActions()` is the source of truth for supported commands.
- `ParseAction(raw)` validates action values before command execution.
- `Executor.Execute(Command)` dispatches using the same internal binding table as
  `SupportedActions()`.
- `DescribeAction(Command)` must always remain aligned to supported actions.

## Compatibility invariants
- Adding/removing actions requires updating `actionBindings`, `SupportedActions`,
  and `DescribeAction` mapping in the same change.
- Every supported action must be parseable and executable through `Executor`.
- Unsupported action inputs must return `UnsupportedActionError`.
- `UnsupportedActionError` message must remain user-action specific.

## Test anchors
Any change to this package boundary must keep existing contract tests green and update them
when needed:
- `goseg/uploadsvc/service_test.go`
  - `TestExecutorDispatchTableParityAcrossSupportedActions`
  - `TestExecutorSupportedActionsMatchesContract`
  - `TestParseActionMatchesSupportedActions`
  - `TestExecutorReturnsUnsupportedActionError`
- Transport adapters should continue to enforce translation integrity:
  - `goseg/uploadsvc/adapters/upload_payload_test.go`

## Integration expectations
- Keep transport-to-domain translation in `goseg/uploadsvc/adapters`.
- The websocket handler (`goseg/handler/ws/upload.go`) consumes commands via this package and
  should not perform action parsing independently.
