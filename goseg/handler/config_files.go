package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/structs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type configFileRequest struct {
	Token   structs.WsTokenStruct `json:"token"`
	Action  string                `json:"action"`
	File    string                `json:"file"`
	Content string                `json:"content"`
}

type configFileSummary struct {
	File  string `json:"file"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
	Pier  string `json:"pier,omitempty"`
}

type configFileResponse struct {
	OK      bool                `json:"ok"`
	Files   []configFileSummary `json:"files,omitempty"`
	File    string              `json:"file,omitempty"`
	Content string              `json:"content,omitempty"`
	Error   string              `json:"error,omitempty"`
}

type configFileTarget struct {
	file string
	kind string
	pier string
	path string
}

var hermesEnvLinePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

func ConfigFilesHandler(w http.ResponseWriter, r *http.Request) {
	setConfigFilesCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeConfigFilesError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	defer r.Body.Close()
	var req configFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeConfigFilesError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %v", err))
		return
	}
	if !authorizeConfigFilesRequest(w, r, req.Token) {
		return
	}
	switch req.Action {
	case "list":
		writeConfigFilesJSON(w, http.StatusOK, configFileResponse{OK: true, Files: listConfigFiles()})
	case "read":
		target, err := resolveConfigFileTarget(req.File)
		if err != nil {
			writeConfigFilesError(w, http.StatusBadRequest, err)
			return
		}
		var content []byte
		switch target.kind {
		case "hermes-yaml":
			content, err = docker.ReadFileFromVolume(docker.HermesDataVolumeName, "config.yaml")
		case "hermes-env":
			content, err = docker.ReadFileFromVolume(docker.HermesDataVolumeName, ".env")
		default:
			content, err = os.ReadFile(target.path)
		}
		if err != nil {
			writeConfigFilesError(w, http.StatusInternalServerError, fmt.Errorf("unable to read %s: %v", target.file, err))
			return
		}
		writeConfigFilesJSON(w, http.StatusOK, configFileResponse{OK: true, File: target.file, Content: string(content)})
	case "save":
		target, err := resolveConfigFileTarget(req.File)
		if err != nil {
			writeConfigFilesError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			writeConfigFilesError(w, http.StatusBadRequest, fmt.Errorf("refusing to save empty config"))
			return
		}
		var formatted []byte
		switch target.kind {
		case "system":
			formatted, err = config.ReplaceConfJSON([]byte(req.Content))
		case "pier":
			formatted, err = config.ReplaceUrbitConfigJSON(target.pier, []byte(req.Content))
		case "hermes-yaml":
			formatted, err = validateHermesConfigYAML([]byte(req.Content))
			if err == nil {
				err = docker.WriteFileToVolume(docker.HermesDataVolumeName, "config.yaml", string(formatted))
			}
		case "hermes-env":
			formatted, err = validateHermesEnv([]byte(req.Content))
			if err == nil {
				err = docker.WriteFileToVolume(docker.HermesDataVolumeName, ".env", string(formatted))
			}
			if err == nil {
				err = docker.WriteFileToVolume(docker.HermesWorkspaceVolumeName, ".env", string(formatted))
			}
		default:
			err = fmt.Errorf("unsupported config kind %q", target.kind)
		}
		if err != nil {
			writeConfigFilesError(w, http.StatusBadRequest, err)
			return
		}
		writeConfigFilesJSON(w, http.StatusOK, configFileResponse{OK: true, File: target.file, Content: string(formatted)})
	default:
		writeConfigFilesError(w, http.StatusBadRequest, fmt.Errorf("unrecognized config file action: %s", req.Action))
	}
}

func listConfigFiles() []configFileSummary {
	conf := config.Conf()
	files := []configFileSummary{
		{
			File:  "system.json",
			Label: "System settings",
			Kind:  "system",
		},
		{
			File:  "hermes/config.yaml",
			Label: "Hermes config.yaml",
			Kind:  "hermes-yaml",
		},
		{
			File:  "hermes/.env",
			Label: "Hermes .env",
			Kind:  "hermes-env",
		},
	}
	for _, pier := range conf.Piers {
		files = append(files, configFileSummary{
			File:  fmt.Sprintf("pier/%s.json", pier),
			Label: fmt.Sprintf("%s pier config", pier),
			Kind:  "pier",
			Pier:  pier,
		})
	}
	return files
}

func resolveConfigFileTarget(file string) (configFileTarget, error) {
	trimmed := strings.TrimSpace(file)
	if trimmed == "" {
		return configFileTarget{}, fmt.Errorf("config file is required")
	}
	cleaned := path.Clean(trimmed)
	if cleaned != trimmed || strings.Contains(cleaned, "..") {
		return configFileTarget{}, fmt.Errorf("invalid config file path")
	}
	switch cleaned {
	case "system.json", "settings.json":
		return configFileTarget{
			file: "system.json",
			kind: "system",
			path: filepath.Join(config.BasePath, "settings", "system.json"),
		}, nil
	case "hermes/config.yaml":
		return configFileTarget{
			file: "hermes/config.yaml",
			kind: "hermes-yaml",
		}, nil
	case "hermes/.env":
		return configFileTarget{
			file: "hermes/.env",
			kind: "hermes-env",
		}, nil
	}
	if path.Dir(cleaned) != "pier" || path.Ext(cleaned) != ".json" {
		return configFileTarget{}, fmt.Errorf("unsupported config file: %s", file)
	}
	pier := strings.TrimSuffix(path.Base(cleaned), ".json")
	if pier == "" || strings.ContainsAny(pier, `/\`) {
		return configFileTarget{}, fmt.Errorf("invalid pier config name")
	}
	if !configuredPier(pier) {
		return configFileTarget{}, fmt.Errorf("pier %s is not configured", pier)
	}
	return configFileTarget{
		file: fmt.Sprintf("pier/%s.json", pier),
		kind: "pier",
		pier: pier,
		path: filepath.Join(config.BasePath, "settings", "pier", pier+".json"),
	}, nil
}

func validateHermesConfigYAML(raw []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("Hermes config.yaml cannot be empty")
	}
	var decoded any
	if err := yaml.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid Hermes config.yaml: %v", err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return nil, fmt.Errorf("Hermes config.yaml must be a YAML mapping")
	}
	return []byte(trimmed + "\n"), nil
}

func validateHermesEnv(raw []byte) ([]byte, error) {
	if strings.ContainsRune(string(raw), '\x00') {
		return nil, fmt.Errorf("Hermes .env cannot contain null bytes")
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("Hermes .env cannot be empty")
	}
	for idx, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		if !hermesEnvLinePattern.MatchString(line) {
			return nil, fmt.Errorf("invalid Hermes .env line %d: expected NAME=value", idx+1)
		}
	}
	return []byte(strings.TrimRight(string(raw), "\r\n") + "\n"), nil
}

func configuredPier(pier string) bool {
	return slices.Contains(config.Conf().Piers, pier)
}

func authorizeConfigFilesRequest(w http.ResponseWriter, r *http.Request, token structs.WsTokenStruct) bool {
	_, valid, authed := auth.CheckStreamToken(token, r)
	if !valid || !authed {
		writeConfigFilesError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return false
	}
	return true
}

func setConfigFilesCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func writeConfigFilesJSON(w http.ResponseWriter, status int, payload configFileResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		zap.L().Error(fmt.Sprintf("failed to write config files response: %v", err))
	}
}

func writeConfigFilesError(w http.ResponseWriter, status int, err error) {
	zap.L().Warn(fmt.Sprintf("config files request failed: %v", err))
	writeConfigFilesJSON(w, status, configFileResponse{OK: false, Error: err.Error()})
}
