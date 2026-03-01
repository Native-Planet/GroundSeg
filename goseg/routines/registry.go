package routines

import routinesystem "groundseg/routines/system"

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

func ChopAtLimit() {
	routinesystem.ChopAtLimit()
}

