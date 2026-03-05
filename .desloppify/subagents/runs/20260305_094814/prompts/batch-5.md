You are a focused subagent reviewer for a single holistic investigation batch.

Repository root: /Users/chuah/NP/GroundSeg
Blind packet: /Users/chuah/NP/GroundSeg/.desloppify/review_packet_blind.json
Batch index: 5
Batch name: Cross-cutting Sweep
Batch dimensions: error_consistency
Batch rationale: selected dimensions had no direct batch mapping; review representative cross-cutting files

Files assigned:
- goseg/protocol/contracts/conformance/fixtures_test.go
- goseg/protocol/contracts/familycatalog/catalog.go
- goseg/protocol/contracts/familyspec/specs_test.go
- goseg/uploadsvc/actions_test.go
- goseg/uploadsvc/doc.go
- goseg/broadcast/state_lifecycle.go
- goseg/broadcast/state_test.go
- goseg/broadcast/state.go
- goseg/broadcast/loop_test.go
- goseg/broadcast/collectors/system_info_test.go
- goseg/rectify/rectify_services.go
- goseg/defaults/defaults.go
- goseg/rectify/rectify_test.go
- goseg/broadcast/loop.go
- goseg/leak/broadcast_test.go
- goseg/docker/orchestration/subsystem/docker_test.go
- goseg/leak/broadcast.go
- goseg/docker/orchestration/wireguard.go
- goseg/docker/orchestration/wireguard_test.go
- goseg/system/wifi_runtime_methods.go
- goseg/leak/leak_test.go
- goseg/broadcast/transitions.go
- goseg/structs/configs.go
- goseg/config/config_test.go
- goseg/config/config_update_registry.go
- goseg/startuporchestrator/startup_orchestrator.go
- goseg/importer/importer.go
- goseg/main.go
- goseg/main_test.go
- goseg/handler/systemsvc/system_test.go
- goseg/docker/orchestration/subsystem/subsystem_events.go
- goseg/routines/version.go
- goseg/startuporchestrator/startup_orchestrator_test.go
- goseg/docker/orchestration/runtime_ops.go
- goseg/docker/events/events.go
- goseg/routines/logstream/logs.go
- goseg/docker/orchestration/subsystem/subsystem_health.go
- goseg/shipworkflow/operations_runtime.go
- goseg/docker/lifecycle/planner.go
- goseg/routines/system/system.go
- goseg/docker/network/network.go
- goseg/system/wifi_runtime_api.go
- goseg/config/runtimecontext/context.go
- goseg/docker/orchestration/internal/artifactwriter/artifact_writer.go
- README.md
- goseg/exporter/exporter_test.go
- goseg/accesspoint/accesspoint_test.go
- goseg/config/config_update_schema.go
- goseg/docker/lifecycle/lifecycle_full_test.go
- goseg/docker/orchestration/runtime_core.go
- goseg/handler/ship/urbit_test.go
- goseg/importer/extract_test.go
- goseg/importer/importer_test.go
- goseg/leak/handler_test.go
- goseg/logger/logger_test.go
- goseg/protocol/actions/actions_test.go
- goseg/protocol/contracts/contracts_test.go
- goseg/session/logstream_session_store.go
- goseg/startram/backup_runtime_test.go
- goseg/startram/state_sync_test.go
- goseg/system/metrics/metrics_test.go
- goseg/system/storage/storage_test.go
- goseg/system/wifi_c2c_flow_test.go
- goseg/system/wifi_test.go
- goseg/uploadsvc/service_test.go

Task requirements:
1. Read the blind packet and follow `system_prompt` constraints exactly.
1a. If previously flagged issues are listed above, use them as context for your review.
    Verify whether each still applies to the current code. Do not re-report fixed or
    wontfix issues. Use them as starting points to look deeper — inspect adjacent code
    and related modules for defects the prior review may have missed.
1c. Think structurally: when you spot multiple individual issues that share a common
    root cause (missing abstraction, duplicated pattern, inconsistent convention),
    explain the deeper structural issue in the finding, not just the surface symptom.
    If the pattern is significant enough, report the structural issue as its own finding
    with appropriate fix_scope ('multi_file_refactor' or 'architectural_change') and
    use `root_cause_cluster` to connect related symptom findings together.
2. Evaluate ONLY listed files and ONLY listed dimensions for this batch.
3. Return 0-10 high-quality findings for this batch (empty array allowed).
3a. Do not suppress real defects to keep scores high; report every material issue you can support with evidence.
3b. Do not default to 100. Reserve 100 for genuinely exemplary evidence in this batch.
4. Score/finding consistency is required: broader or more severe findings MUST lower dimension scores.
4a. Any dimension scored below 85.0 MUST include explicit feedback: add at least one finding with the same `dimension` and a non-empty actionable `suggestion`.
5. Every finding must include `related_files` with at least 2 files when possible.
6. Every finding must include `dimension`, `identifier`, `summary`, `evidence`, `suggestion`, and `confidence`.
7. Every finding must include `impact_scope` and `fix_scope`.
8. Every scored dimension MUST include dimension_notes with concrete evidence.
9. If a dimension score is >85.0, include `issues_preventing_higher_score` in dimension_notes.
10. Use exactly one decimal place for every assessment and abstraction sub-axis score.
11. Ignore prior chat context and any target-threshold assumptions.
12. Do not edit repository files.
13. Return ONLY valid JSON, no markdown fences.

Scope enums:
- impact_scope: "local" | "module" | "subsystem" | "codebase"
- fix_scope: "single_edit" | "multi_file_refactor" | "architectural_change"

Output schema:
{
  "batch": "Cross-cutting Sweep",
  "batch_index": 5,
  "assessments": {"<dimension>": <0-100 with one decimal place>},
  "dimension_notes": {
    "<dimension>": {
      "evidence": ["specific code observations"],
      "impact_scope": "local|module|subsystem|codebase",
      "fix_scope": "single_edit|multi_file_refactor|architectural_change",
      "confidence": "high|medium|low",
      "issues_preventing_higher_score": "required when score >85.0",
      "sub_axes": {"abstraction_leverage": 0-100 with one decimal place, "indirection_cost": 0-100 with one decimal place, "interface_honesty": 0-100 with one decimal place}  // required for abstraction_fitness when evidence supports it
    }
  },
  "findings": [{
    "dimension": "<dimension>",
    "identifier": "short_id",
    "summary": "one-line defect summary",
    "related_files": ["relative/path.py"],
    "evidence": ["specific code observation"],
    "suggestion": "concrete fix recommendation",
    "confidence": "high|medium|low",
    "impact_scope": "local|module|subsystem|codebase",
    "fix_scope": "single_edit|multi_file_refactor|architectural_change",
    "root_cause_cluster": "optional_cluster_name_when_supported_by_history"
  }],
  "retrospective": {
    "root_causes": ["optional: concise root-cause hypotheses"],
    "likely_symptoms": ["optional: identifiers that look symptom-level"],
    "possible_false_positives": ["optional: prior concept keys likely mis-scoped"]
  }
}
