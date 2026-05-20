package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func GetWorkingDirectory() string {
	hd, _ := os.UserHomeDir()
	return path.Join(hd, ".httpgo/collections")
}

func GetGlobalEnvFile() string {
	return path.Join(GetWorkingDirectory(), "globalenv")
}

// Load reads each fileName as a KEY=value env file and merges the contents
// into a single map. Later files override earlier ones on key conflicts.
func Load(fileNames ...string) (map[string]string, error) {
	envs := map[string]string{}

	for _, fileName := range fileNames {
		fileData, err := os.ReadFile(filepath.Clean(fileName))
		if err != nil {
			return nil, err
		}

		fileEnvs, err := parse(fileData, true)
		if err != nil {
			return nil, fmt.Errorf("failed to parse file at %s: %w", fileName, err)
		}

		for key, value := range fileEnvs {
			envs[key] = value
		}
	}

	return envs, nil
}

var (
	skipLine      = regexp.MustCompile(`^\s*#|^\s*"#`)
	quotedSegment = regexp.MustCompile(`"(.*?)"`)
	inlineComment = regexp.MustCompile(`#.*`)
)

func parse(data []byte, stripQuotes bool) (map[string]string, error) {
	envs := map[string]string{}

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || skipLine.MatchString(line) {
			continue
		}

		lineParts := strings.SplitN(line, "=", 2)
		if len(lineParts) != 2 {
			return nil, fmt.Errorf("failed to parse line, expected 2 parts got %d", len(lineParts))
		}

		key := strings.TrimSpace(lineParts[0])
		value := strings.TrimSpace(lineParts[1])

		// If the value contains a quoted segment, keep just that segment;
		// otherwise drop any "# inline comment" tail.
		if match := quotedSegment.FindString(value); match != "" {
			value = match
		} else {
			value = strings.TrimSpace(inlineComment.ReplaceAllString(value, ""))
		}

		if stripQuotes && len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		envs[key] = value
	}

	return envs, nil
}
