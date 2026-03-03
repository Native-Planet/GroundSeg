package system

import "groundseg/system/storage"

type fstabRecord struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
	Dump       string
	Pass       string
}

func (record fstabRecord) line() string {
	return storage.FstabRecord{
		Device:     record.Device,
		MountPoint: record.MountPoint,
		FSType:     record.FSType,
		Options:    record.Options,
		Dump:       record.Dump,
		Pass:       record.Pass,
	}.Line()
}

func parseFstabLine(raw string) (fstabRecord, bool) {
	record, ok := storage.ParseFstabLine(raw)
	if !ok {
		return fstabRecord{}, false
	}
	return fstabRecord{
		Device:     record.Device,
		MountPoint: record.MountPoint,
		FSType:     record.FSType,
		Options:    record.Options,
		Dump:       record.Dump,
		Pass:       record.Pass,
	}, true
}

func reconcileFstabLines(lines []string, desired fstabRecord) ([]string, bool) {
	return storage.ReconcileFstabLines(lines, storage.FstabRecord{
		Device:     desired.Device,
		MountPoint: desired.MountPoint,
		FSType:     desired.FSType,
		Options:    desired.Options,
		Dump:       desired.Dump,
		Pass:       desired.Pass,
	})
}

func readFstabLines(path string) ([]string, error) {
	return storage.ReadFstabLines(path, resolveDiskSeams())
}

func writeFstabLines(path string, lines []string) error {
	return storage.WriteFstabLines(path, lines, resolveDiskSeams())
}
