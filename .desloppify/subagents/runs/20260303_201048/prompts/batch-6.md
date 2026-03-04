You are a focused subagent reviewer for a single holistic investigation batch.

Repository root: /Users/chuah/NP/GroundSeg
Blind packet: /Users/chuah/NP/GroundSeg/.desloppify/review_packet_blind.json
Batch index: 6
Batch name: Full Codebase Sweep
Batch dimensions: cross_module_architecture, error_consistency, abstraction_fitness, test_strategy, design_coherence
Batch rationale: thorough default: evaluate cross-cutting quality across all production files

Files assigned:
- goseg/accesspoint/accesspoint.go
- goseg/accesspoint/config.go
- goseg/accesspoint/lifecycle_coordinator.go
- goseg/accesspoint/parameters.go
- goseg/accesspoint/router.go
- goseg/auth/auth.go
- goseg/auth/client_session.go
- goseg/auth/lifecycle/lifecycle.go
- goseg/auth/tokens/tokens.go
- goseg/authsession/session.go
- goseg/backups/backups.go
- goseg/backupsvc/service.go
- goseg/broadcast/broadcast.go
- goseg/broadcast/collectors/app_profile_system_collectors.go
- goseg/broadcast/collectors/collectors_backup.go
- goseg/broadcast/collectors/collectors_facade.go
- goseg/broadcast/collectors/collectors_runtime.go
- goseg/broadcast/collectors/collectors_startram.go
- goseg/broadcast/collectors/collectors_urbit.go
- goseg/broadcast/delivery.go
- goseg/broadcast/loop.go
- goseg/broadcast/state.go
- goseg/broadcast/transitions.go
- goseg/chopsvc/service.go
- goseg/click/acme/service.go
- goseg/click/backup/service.go
- goseg/click/click.go
- goseg/click/desk/desk.go
- goseg/click/internal/response/response.go
- goseg/click/internal/runtime/runtime.go
- goseg/click/lifecycle/lifecycle.go
- goseg/click/luscode/luscode.go
- goseg/click/notify/notify.go
- goseg/click/pack/pack.go
- goseg/click/restore/restore.go
- goseg/click/storage/storage.go
- goseg/config/bootstrap.go
- goseg/config/config.go
- goseg/config/config_events.go
- goseg/config/config_merge.go
- goseg/config/config_state.go
- goseg/config/config_update.go
- goseg/config/config_update_executor.go
- goseg/config/config_update_options.go
- goseg/config/config_update_registry.go
- goseg/config/config_update_schema.go
- goseg/config/config_view.go
- goseg/config/mc.go
- goseg/config/minio.go
- goseg/config/netdata.go
- goseg/config/persistence.go
- goseg/config/storage.go
- goseg/config/urbit.go
- goseg/config/version.go
- goseg/config/wireguard.go
- goseg/config/wireguardbuilder/builder.go
- goseg/config/wireguardkeys/keys.go
- goseg/config/wireguardstore/store.go
- goseg/defaults/defaults.go
- goseg/defaults/scripts.go
- goseg/defaults/version.go
- goseg/docker/docker.go
- goseg/docker/events/events.go
- goseg/docker/lifecycle/commands.go
- goseg/docker/lifecycle/containers.go
- goseg/docker/lifecycle/lifecycle.go
- goseg/docker/lifecycle/planner.go
- goseg/docker/lifecycle/poller.go
- goseg/docker/lifecycle/queries.go
- goseg/docker/lifecycle/status_query.go
- goseg/docker/network/network.go
- goseg/docker/orchestration/container/container_runtime.go
- goseg/docker/orchestration/container/llama.go
- goseg/docker/orchestration/container/minio.go
- goseg/docker/orchestration/container/netdata.go
- goseg/docker/orchestration/container_bridge.go
- goseg/docker/orchestration/container_config.go
- goseg/docker/orchestration/internal/artifactwriter/artifact_writer.go
- goseg/docker/orchestration/llama.go
- goseg/docker/orchestration/minio.go

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
  "batch": "Full Codebase Sweep",
  "batch_index": 6,
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
