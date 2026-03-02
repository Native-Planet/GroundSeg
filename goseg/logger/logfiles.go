package logger

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type partedLogFile struct {
	path string
	date string
	part int
}

func parsePartedLogFileName(name string) (date string, part int, ok bool) {
	if !strings.HasSuffix(name, ".log") {
		return "", 0, false
	}
	parts := strings.SplitN(name, "-part-", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	part, err := strconv.Atoi(strings.TrimSuffix(parts[1], ".log"))
	if err != nil {
		return "", 0, false
	}
	return parts[0], part, true
}

func MostRecentPartedLogPaths(dirPath string, files []os.FileInfo, keep int) []string {
	if keep <= 0 {
		return nil
	}
	logFiles := make([]partedLogFile, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}
		date, part, ok := parsePartedLogFileName(file.Name())
		if !ok {
			continue
		}
		logFiles = append(logFiles, partedLogFile{path: filepath.Join(dirPath, file.Name()), date: date, part: part})
	}

	sort.Slice(logFiles, func(i, j int) bool {
		if logFiles[i].date == logFiles[j].date {
			return logFiles[i].part > logFiles[j].part
		}
		return logFiles[i].date > logFiles[j].date
	})

	if len(logFiles) > keep {
		logFiles = logFiles[:keep]
	}
	recentFiles := make([]string, 0, len(logFiles))
	for _, file := range logFiles {
		recentFiles = append(recentFiles, file.path)
	}
	return recentFiles
}
