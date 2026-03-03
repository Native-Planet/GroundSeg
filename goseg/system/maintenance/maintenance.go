package maintenance

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"groundseg/defaults"
)

// IsNPBox returns true when basePath/natively planet marker exists.
func IsNPBox(basePath string) bool {
	filePath := filepath.Join(basePath, "nativeplanet")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// FixerScript installs and enables periodic fixer script if this is an NP box.
func FixerScript(basePath string) error {
	// check if it's one of our boxes
	if !IsNPBox(basePath) {
		return nil
	}

	// Create fixer.sh
	zap.L().Info("Thank you for supporting Native Planet!")
	fixer := filepath.Join(basePath, "fixer.sh")
	if _, err := os.Stat(fixer); os.IsNotExist(err) {
		zap.L().Info("Fixer script not detected, creating")
		err := ioutil.WriteFile(fixer, []byte(defaults.Fixer), 0755)
		if err != nil {
			return fmt.Errorf("create fixer script %q: %w", fixer, err)
		}
	}

	// make it a cron
	if !CronExists(fixer) {
		zap.L().Info("Fixer cron not found, creating")
		cronJob := fmt.Sprintf("*/5 * * * * /bin/bash %s\n", fixer)
		if err := AddCron(cronJob); err != nil {
			return fmt.Errorf("setup fixer cron: %w", err)
		}
	} else {
		zap.L().Info("Fixer cron found. Doing nothing")
	}
	return nil
}

func CronExists(fixerPath string) bool {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return false
	}
	outStr := string(out)
	return strings.Contains(outStr, fixerPath) && strings.Contains(outStr, "/bin/bash")
}

func AddCron(job string) error {
	tmpfile, err := ioutil.TempFile("", "cron")
	if err != nil {
		return fmt.Errorf("create temporary crontab file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || !strings.Contains(string(exitErr.Stderr), "no crontab for") {
			return fmt.Errorf("read existing crontab: %w", err)
		}
		out = []byte{}
	}

	if _, err := tmpfile.WriteString(string(out)); err != nil {
		return fmt.Errorf("write existing crontab to temp file: %w", err)
	}
	if _, err := tmpfile.WriteString(job); err != nil {
		return fmt.Errorf("append cron job to temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("close crontab temp file: %w", err)
	}

	cmd := exec.Command("crontab", tmpfile.Name())
	return cmd.Run()
}
