// Package system exposes handler-facing host operations for storage,
// maintenance, metrics, and Wi-Fi control.
//
// System operations intentionally sequence privileged side effects through
// runtime seams so tests can assert behavior without mutating host state.
package system
