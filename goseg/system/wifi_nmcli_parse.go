package system

import (
	"errors"
	"fmt"
	"strings"
)

var ErrWiFiPartialResult = errors.New("wifi command returned partial results")

type wifiNmcliDevice struct {
	name string
	typ  string
}

type wifiNmcliScanResult struct {
	ssid string
}

type wifiNmcliParseRecord struct {
	index int
	line  string
	data  []string
}

type wifiNmcliParseError struct {
	record    wifiNmcliParseRecord
	required  int
	found     int
	operation string
}

type wifiNmcliPartialParseError struct {
	operation   string
	droppedRows int
	first       error
}

type wifiPartialResultWarning struct {
	operation string
	cause     error
}

func (e wifiNmcliParseError) Error() string {
	return "invalid nmcli output for " + e.operation + "; expected " +
		fmt.Sprintf("%d fields", e.required) + ", found " + fmt.Sprintf("%d", e.found)
}

func (e wifiNmcliPartialParseError) Error() string {
	return fmt.Sprintf("nmcli %s parsed with %d malformed row(s): %v", e.operation, e.droppedRows, e.first)
}

func (e wifiNmcliPartialParseError) Unwrap() error {
	return e.first
}

func (e wifiPartialResultWarning) Error() string {
	return fmt.Sprintf("%s returned partial results: %v", e.operation, e.cause)
}

func (e wifiPartialResultWarning) Unwrap() error {
	return e.cause
}

func (e wifiPartialResultWarning) Is(target error) bool {
	return target == ErrWiFiPartialResult
}

func wrapWiFiPartialResult(operation string, err error) error {
	if err == nil {
		return nil
	}
	var partialErr wifiNmcliPartialParseError
	if !errors.As(err, &partialErr) {
		return err
	}
	return wifiPartialResultWarning{
		operation: operation,
		cause:     partialErr,
	}
}

func newWiFiNmcliParseError(operation, line string, index, required, found int) wifiNmcliParseError {
	return wifiNmcliParseError{
		record: wifiNmcliParseRecord{
			index: index,
			line:  line,
			data:  splitNmcliRecord(line),
		},
		required:  required,
		found:     found,
		operation: operation,
	}
}

func splitNmcliRecord(raw string) []string {
	fields := make([]string, 0, 8)
	var field strings.Builder
	escaped := false
	for _, char := range raw {
		switch {
		case escaped:
			field.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == ':':
			fields = append(fields, strings.TrimSpace(field.String()))
			field.Reset()
		default:
			field.WriteRune(char)
		}
	}
	fields = append(fields, strings.TrimSpace(field.String()))
	return fields
}

func parseNmcliWifiDeviceRecord(raw string, index int) (wifiNmcliDevice, error) {
	fields := splitNmcliRecord(raw)
	if len(fields) < 2 {
		return wifiNmcliDevice{}, newWiFiNmcliParseError(
			"wifi device list", raw, index, 2, len(fields),
		)
	}
	return wifiNmcliDevice{name: strings.TrimSpace(fields[0]), typ: strings.TrimSpace(fields[1])}, nil
}

func parseNmcliWifiScanResultRecord(raw string, index int) (wifiNmcliScanResult, error) {
	fields := splitNmcliRecord(raw)
	if len(fields) <= 7 {
		return wifiNmcliScanResult{}, newWiFiNmcliParseError(
			"wifi scan result", raw, index, 8, len(fields),
		)
	}
	return wifiNmcliScanResult{ssid: strings.TrimSpace(fields[7])}, nil
}

func parseNmcliWifiDeviceRecords(raw string) ([]string, error) {
	rawLines := strings.Split(strings.TrimSpace(raw), "\n")
	devices := make([]string, 0, len(rawLines))
	parseErrors := make([]error, 0)
	for index, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		deviceRecord, err := parseNmcliWifiDeviceRecord(trimmed, index)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		if deviceRecord.typ == "wifi" && deviceRecord.name != "" {
			devices = append(devices, deviceRecord.name)
		}
	}
	if len(parseErrors) > 0 {
		if len(devices) == 0 {
			return nil, parseErrors[0]
		}
		return devices, wifiNmcliPartialParseError{
			operation:   "wifi device list",
			droppedRows: len(parseErrors),
			first:       parseErrors[0],
		}
	}
	return devices, nil
}

func parseNmcliWifiScanResults(raw string) ([]wifiNmcliScanResult, error) {
	rawLines := strings.Split(strings.TrimSpace(raw), "\n")
	results := make([]wifiNmcliScanResult, 0, len(rawLines))
	parseErrors := make([]error, 0)
	for index, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		record, err := parseNmcliWifiScanResultRecord(trimmed, index)
		if err != nil {
			parseErrors = append(parseErrors, err)
			continue
		}
		if record.ssid == "" {
			continue
		}
		results = append(results, record)
	}
	if len(parseErrors) > 0 {
		if len(results) == 0 {
			return nil, parseErrors[0]
		}
		return results, wifiNmcliPartialParseError{
			operation:   "wifi scan result list",
			droppedRows: len(parseErrors),
			first:       parseErrors[0],
		}
	}
	return results, nil
}

func parseNmcliWifiDevices(raw string) ([]string, error) {
	parsedDevices, err := parseNmcliWifiDeviceRecords(raw)
	if len(parsedDevices) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, ErrWiFiInterfaceNotFound
	}
	return parsedDevices, err
}

func parseNmcliWifiSSIDs(raw string) ([]string, error) {
	scans, err := parseNmcliWifiScanResults(raw)
	ssids := make([]string, 0, len(scans))
	for _, scan := range scans {
		ssids = append(ssids, scan.ssid)
	}
	if err != nil && len(ssids) == 0 {
		return nil, err
	}
	return ssids, err
}
