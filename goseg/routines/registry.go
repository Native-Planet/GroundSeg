package routines

import (
	"context"

	routinesystem "groundseg/routines/system"
)

func StartBackupRoutines() {
	routinesystem.StartBackupRoutines()
}

func TlonBackupRemote() {
	routinesystem.TlonBackupRemote()
}

func TlonBackupLocal() {
	routinesystem.TlonBackupLocal()
}

func StartChopRoutines() {
	routinesystem.StartChopRoutines()
}

func StartChopRoutinesWithContext(ctx context.Context) error {
	return routinesystem.StartChopRoutinesWithContext(ctx)
}

func ChopAtLimit() {
	routinesystem.ChopAtLimit()
}

func StartBackupRoutinesWithContext(ctx context.Context) error {
	return routinesystem.StartBackupRoutinesWithContext(ctx)
}
