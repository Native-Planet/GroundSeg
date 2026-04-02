package docker

import (
	"fmt"
	"groundseg/structs"
	"strings"
	"unicode"
)

type flagRule struct {
	canonical string
	aliases   []string
}

var (
	persistentExtraArgRules = []flagRule{
		{canonical: "-B/--bootstrap", aliases: []string{"-B", "--bootstrap"}},
		{canonical: "-c/--pier", aliases: []string{"-c", "--pier"}},
		{canonical: "-d/--daemon", aliases: []string{"-d", "--daemon"}},
		{canonical: "-G/--key-string", aliases: []string{"-G", "--key-string"}},
		{canonical: "-p/--port/--ames-port", aliases: []string{"-p", "--port", "--ames-port"}},
		{canonical: "--http-port", aliases: []string{"--http-port"}},
		{canonical: "--loom", aliases: []string{"--loom"}},
		{canonical: "--snap-time", aliases: []string{"--snap-time"}},
		{canonical: "--dirname", aliases: []string{"--dirname"}},
		{canonical: "--devmode", aliases: []string{"--devmode"}},
		{canonical: "-t", aliases: []string{"-t"}},
		{canonical: "-w/--name", aliases: []string{"-w", "--name"}},
		{canonical: "--bootstrap-url", aliases: []string{"--bootstrap-url"}},
		{canonical: "--prop-url", aliases: []string{"--prop-url"}},
		{canonical: "--prop-name", aliases: []string{"--prop-name"}},
	}
	firstBootArgRules = []flagRule{
		{canonical: "-G/--key-string", aliases: []string{"-G", "--key-string"}},
		{canonical: "-k/--key-file", aliases: []string{"-k", "--key-file"}},
		{canonical: "-p/--port/--ames-port", aliases: []string{"-p", "--port", "--ames-port"}},
		{canonical: "--http-port", aliases: []string{"--http-port"}},
		{canonical: "--loom", aliases: []string{"--loom"}},
		{canonical: "-t", aliases: []string{"-t"}},
		{canonical: "-w/--name", aliases: []string{"-w", "--name"}},
		{canonical: "-x", aliases: []string{"-x"}},
	}
)

type UrbitBootCommand struct {
	ScriptArgs  []string
	PreviewBase string
	PreviewFull string
}

func BuildUrbitBootCommand(shipConf structs.UrbitDocker, systemConf structs.SysConfig, bootStatus string) (UrbitBootCommand, error) {
	loomValue := fmt.Sprintf("%v", shipConf.LoomSize)
	devMode := "False"
	if shipUsesDevMode(bootStatus) && shipConf.DevMode {
		devMode = "True"
	}

	snapTime := resolvedSnapTime(systemConf, shipConf)
	httpPort, amesPort := resolvedRuntimePorts(shipConf)

	scriptArgs := []string{
		"bash",
		"/urbit/start_urbit.sh",
		"--loom=" + loomValue,
		"--dirname=" + shipConf.PierName,
		"--devmode=" + devMode,
	}
	if shipConf.Network == "wireguard" {
		scriptArgs = append(scriptArgs,
			"--http-port="+httpPort,
			"--port="+amesPort,
		)
	}
	scriptArgs = append(scriptArgs, "--snap-time="+snapTime)

	extraArgs, err := ValidatePersistentExtraArgs(shipConf.ExtraArgs)
	if err != nil {
		return UrbitBootCommand{}, fmt.Errorf("invalid extra args: %w", err)
	}
	scriptArgs = append(scriptArgs, extraArgs...)

	baseArgs := []string{
		"-p", amesPort,
		"--http-port", httpPort,
		"--loom", loomValue,
		"--snap-time", snapTime,
		shipConf.PierName,
	}
	fullArgs := append(append([]string{}, baseArgs...), extraArgs...)

	return UrbitBootCommand{
		ScriptArgs:  scriptArgs,
		PreviewBase: formatShellCommand("urbit", baseArgs),
		PreviewFull: formatShellCommand("urbit", fullArgs),
	}, nil
}

func ParseExtraArgs(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	var (
		args         []string
		current      strings.Builder
		tokenStarted bool
		inSingle     bool
		inDouble     bool
		escaped      bool
	)

	flush := func() {
		if !tokenStarted {
			return
		}
		args = append(args, current.String())
		current.Reset()
		tokenStarted = false
	}

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case inSingle:
			if r == '\'' {
				inSingle = false
			} else {
				current.WriteRune(r)
			}
		case inDouble:
			switch r {
			case '"':
				inDouble = false
			case '\\':
				escaped = true
			default:
				current.WriteRune(r)
			}
		default:
			switch {
			case unicode.IsSpace(r):
				flush()
			case r == '\'':
				inSingle = true
				tokenStarted = true
			case r == '"':
				inDouble = true
				tokenStarted = true
			case r == '\\':
				escaped = true
				tokenStarted = true
			default:
				current.WriteRune(r)
				tokenStarted = true
			}
		}
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape in extra args")
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote in extra args")
	}
	flush()
	return args, nil
}

func ValidatePersistentExtraArgs(input string) ([]string, error) {
	args, err := ParseExtraArgs(input)
	if err != nil {
		return nil, err
	}
	if err := validateArgsAgainstRules(args, persistentExtraArgRules); err != nil {
		return nil, err
	}
	return args, nil
}

func ValidateFirstBootArgs(input string) ([]string, error) {
	args, err := ParseExtraArgs(input)
	if err != nil {
		return nil, err
	}
	if err := validateArgsAgainstRules(args, firstBootArgRules); err != nil {
		return nil, err
	}
	return args, nil
}

func validateArgsAgainstRules(args []string, rules []flagRule) error {
	var blocked []string
	seen := make(map[string]struct{})
	for _, arg := range args {
		for _, rule := range rules {
			if !matchesAnyAlias(arg, rule.aliases) {
				continue
			}
			if _, exists := seen[rule.canonical]; exists {
				break
			}
			seen[rule.canonical] = struct{}{}
			blocked = append(blocked, rule.canonical)
			break
		}
	}
	if len(blocked) > 0 {
		return fmt.Errorf("not allowed here: %s", strings.Join(blocked, ", "))
	}
	return nil
}

func matchesAnyAlias(token string, aliases []string) bool {
	for _, alias := range aliases {
		if tokenMatchesFlag(token, alias) {
			return true
		}
	}
	return false
}

func tokenMatchesFlag(token string, flag string) bool {
	if token == flag || strings.HasPrefix(token, flag+"=") {
		return true
	}
	if strings.HasPrefix(flag, "--") {
		return false
	}
	return len(token) > len(flag) && strings.HasPrefix(token, flag)
}

func resolvedSnapTime(systemConf structs.SysConfig, shipConf structs.UrbitDocker) string {
	snapTime := "60"
	if systemConf.SnapTime != 0 && systemConf.SnapTime != 60 {
		snapTime = fmt.Sprintf("%v", systemConf.SnapTime)
	}
	if shipConf.SnapTime != 0 && shipConf.SnapTime != 60 {
		snapTime = fmt.Sprintf("%v", shipConf.SnapTime)
	}
	return snapTime
}

func resolvedRuntimePorts(shipConf structs.UrbitDocker) (string, string) {
	if shipConf.Network == "wireguard" {
		return fmt.Sprintf("%v", shipConf.WgHTTPPort), fmt.Sprintf("%v", shipConf.WgAmesPort)
	}
	return "80", "34343"
}

func formatShellCommand(command string, args []string) string {
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-._/:=@,+", r))
	}) == -1 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
}
