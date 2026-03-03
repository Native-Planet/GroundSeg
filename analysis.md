Issue 1 investigation: runtime seam-sprawl and manual callback projection.

Observed in:
- goseg/docker/orchestration/runtime_ops.go
- goseg/docker/orchestration/runtime_core.go
- goseg/broadcast/collectors/collectors_runtime.go
- goseg/docker/orchestration/container/netdata.go
- goseg/docker/orchestration/container_bridge.go

Findings:
- `collectorRuntime` composes two nested callback structs but still relies on hand-written `mergeCollector*Runtime` functions that duplicate field-by-field merge logic.
- Container/bridge seam projection still assigns every `NetdataRuntime` field manually in `applyNetdataRuntime`, increasing drift risk when adding new seams.
- This pattern is repeated across runtime layers and is a good candidate for a shared, reflective merge strategy already present in `groundseg/internal/seams`.

Proposed fix:
- Replace manual collector runtime merge with `seams.Merge` to avoid per-field override boilerplate and keep default wiring and override wiring mechanically aligned.
- Add a dedicated Netdata runtime bridge composition path (`netdataRuntimeOps`) and apply overrides via `seams.Merge` so adding/removing dependencies is localized to the ops definition, not multiple manual assignments.
- Preserve behavior by keeping default runtime selection unchanged and keeping explicit fallback behavior in callers/tests.
