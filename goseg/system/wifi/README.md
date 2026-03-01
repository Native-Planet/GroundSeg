# Wi-Fi System Gateway Package

This package owns package-local governance for Wi-Fi contract boundaries. High-level
rollout and architecture notes remain in root docs; this file is authoritative for
local compatibility invariants.

## Sub-boundaries
- `goseg/system/wifi/transport`: websocket adapter and broadcast mechanics.
- `goseg/system/wifi/service`: C2C action contract and execution semantics.
- `goseg/system/wifi_transport_adapter.go` in `goseg/system` provides adapter wiring.

## C2C action contract
- Package `system/wifi/service` consumes shared action enums from
  `groundseg/protocol/actions` and keeps parser/executor binding aligned:
  - `ParseC2CAction(raw)` accepts only supported values from
    `protocol/actions`.
  - `c2cActionBindings` and `C2CService.Execute` must remain aligned.
  - Unsupported action values return shared `UnsupportedActionError`.
- Adapter glue in `goseg/system/wifi_c2c_service.go` should remain a simple dependency
  injection boundary and delegate to service contracts.

## Integration expectations
- `processC2CMessageForAdapter` in `goseg/system/wifi_c2c_service.go` must continue
  to decode message payloads and dispatch via package service execution semantics.
- `goseg/system/wifi_radio_service.go` owns wifi hardware actions and should only
  provide adapter-level effects.
- Keep websocket transport concerns isolated from business logic:
  - transport parsing, connection handling, and broadcasting stays in `system/wifi/transport`.
  - command parsing and action execution stays in `system/wifi/service`.

## Test anchors
- `goseg/system/wifi_test.go`
  - `TestParseC2CAction`
  - `TestParseC2CActionRejectsUnsupportedAction`
  - `TestProcessMessageRejectsUnsupportedAction`
  - `TestProcessC2CMessageForAdapter`
  - `TestC2CActionBindingsCoverKnownActions`

## Review expectations
- Any new Wi-Fi action must be added to `groundseg/protocol/actions`, service bindings,
  parser tests, and adapter integration tests in the same change.
