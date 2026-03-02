## Issue 1 Analysis: dc3_volume_bootstrap_pattern_split

- Investigated orchestration volume bootstrapping paths in `netdata.go`, `wireguard.go`, `minio.go`, and `llama.go`.
- Implemented a shared bootstrap primitive in `volume_writer.go`:
  - Added `VolumeInitializationPlan` with explicit volume creation semantics.
  - Centralized copy/write fallback flow in `writeConfigArtifact`.
  - Made volume existence/creation conditional and shared with copy fallback.
- Wired runtime-specific plan configuration through existing `dockerRuntime` seam:
  - `netdata`, `wireguard`, and `minio` now use runtime-provided volume operations for bootstrapping.
  - Llama now uses the same plan constructor via its existing helper.
- Updated tests to use deterministic no-op volume ops in runtime seams where Docker is not available.

## Issue 2 Analysis: 8bd44875

- Reviewed `backup_runtime.go` and identified transport, crypto, and restore-runtime responsibilities coexisting in one file.
- Split runtime behavior into focused files:
  - `startram/backup_transport.go` now owns API call and download verification logic.
  - `startram/backup_crypto.go` now owns AES-GCM encrypt/decrypt helpers.
  - `startram/backup_restore_runtime.go` now owns restore request types and restore-runtime orchestration/volume replay flow.
- Kept orchestration boundary in `backup_service.go` unchanged (interface + orchestration entrypoints) and retained existing public functions by moving implementations out of `backup_runtime.go`.
- Added compatibility coordinator in `backup_runtime.go` to document the new split and avoid a monolithic mixed-concern module.
## Issue 8f Issue 1 Analysis: desk_lifecycle_duplication

- Investigated `goseg/shipworkflow/urbit_operations.go` where install/remove desk handlers for `penpai` and `groundseg` duplicated status/action/wait orchestration.
- The shared flow is: read desk state, branch install/revive/wait for install path, uninstall, then wait for expected running state change.
- Confirmed this flow is currently wrapped only by `runDeskTransition` and duplicated across 4 functions.
- I will replace this with a generic desk lifecycle helper that owns the shared transition flow and accepts per-desk parameters (desk name and operation callbacks), reducing duplicate code and adding a central place for action messages.
## Issue 2 Analysis: routines-capability-blend

- Investigated `routines/disk.go` and `routines/logs.go` against the current finding and confirmed mixed domain responsibilities (disk health + log transport/streaming).
- Implemented domain split under `routines/healthcheck` and `routines/logstream`:
  - Moved existing disk polling/state logic and notification helpers into `routines/healthcheck`.
  - Moved legacy log cleanup/splitting, websocket streaming, and session cleanup logic into `routines/logstream`.
- Replaced `routines` root `disk.go`/`logs.go` with thin delegation functions to preserve exported API used by startup orchestration.
- Relocated corresponding tests to their new packages and kept behavior-equivalent coverage.
## Issue 3 Analysis: bcast_state_loss_on_collector_error

- Investigated `runBroadcastTickWithRuntime` and confirmed the pier collector error path currently allowed `pierInfo` to become empty, which then overwrote in-memory urbit state for that tick.
- Changed failure behavior in `broadcast/loop.go` to retain the last-known `broadcastState.Urbits` when `constructPierInfoFn` returns an error.
- Added regression coverage in `broadcast/loop_test.go` to assert `Transition` and `Info` for existing urbits are preserved across a simulated collector failure.

## Issue 4 Analysis: startup_optional_runtime_handoffs

- Investigated startup runtime handoff in `goseg/startup_orchestrator.go`.
- Introduced explicit runtime contract validation:
  - Added required-callback metadata and validation checks for `startupRuntime`.
  - Added required callback checks for `startBackgroundServicesRuntime`, with `syncRetrieve` conditionally required when `startramWgRegistered == true`.
- Updated bootstrap path:
  - `Bootstrap` now validates the merged startup runtime before task execution.
  - `startBackgroundServicesWithRuntime` now validates background callbacks and returns an error on missing required hooks.
  - Required background callbacks are invoked without silent nil-guarding after validation.
- Added/updated unit tests in `goseg/startup_orchestrator_test.go` to cover:
  - missing startup runtime callbacks rejection,
  - missing background runtime callbacks rejection,
  - expected callback startup sequencing when required hooks are supplied,
  - no-op behavior when startram sync is intentionally disabled.

## Issue 5 Analysis: urbit_conf_snapshot_stale_mutation

- Investigated `urbitContainerConfWithRuntime` in `goseg/docker/orchestration/urbit.go`.
- Fixed stale in-memory config usage by refreshing effective ship config after `UpdateUrbit` writes.
- Switched subsequent container decision logic (`BootStatus`, script path, ports, runtime settings, network mode, mounts) to the reloaded config.
- Added regression coverage in `goseg/docker/orchestration/urbit_test.go` to ensure a config mutation that changes `BootStatus` is honored immediately and drives expected script selection.

## Issue 6 Analysis: c2c_killswitch_no_shutdown_boundary

- Investigated C2C check and lifecycle wiring in `goseg/main.go`.
- Changed C2C check entrypoints to accept a context so kill-switch startup can observe shutdown signals.
- Updated kill-switch loop (`killSwitch`) to accept the same context, use timer+select and return when context is canceled.
- Threaded context through startup callback flow (`StartupOptions.StartC2cCheck`, startup orchestrator background startup path, and main bootstrap wiring) so C2C maintenance stops with application shutdown.
- Added regression test in `goseg/main_test.go` verifying `killSwitch` exits promptly when context is done.
- Evidence confirms config.go and version.go still combine configuration loading, mutation patch construction, runtime snapshots, merge logic, and utility helpers.
- I will split boundaries into focused modules while preserving exported API for callers:
  - loader/parsing: reading raw system.json, default config creation, bootstrap merge initialization.
  - schema/evolution: default/merge rules moved into dedicated `configmerge` section with `ConfigMerger` abstraction and tests.
  - persistence: dedicated writer service for in-memory config snapshots and atomic file persistence.
  - update surface: typed patch application moved into dedicated updater module with small option structs and validation.
- I will also add a light `ConfigView` interface with a concrete `LiveConfigView` so callers can receive immutable read-only settings views rather than using `Conf()` directly where possible.
- This will reduce large cross-cutting functions in `config.go` and align with suggested composable module boundaries.
