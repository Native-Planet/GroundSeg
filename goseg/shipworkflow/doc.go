// Package shipworkflow orchestrates ship lifecycle phases (create, recover,
// register, import) and emits urbit transition events for each phase.
//
// Workflow operations must preserve transition ordering (loading -> success or
// error) so clients never observe phase regressions during long-running tasks.
package shipworkflow
