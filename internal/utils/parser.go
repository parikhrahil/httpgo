package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/parikhrahil/httpgo/internal/config"
)

// GetValidCollections returns the names of every namespace directory under dir.
func GetValidCollections(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", dir, err)
	}
	var coll []string
	for _, entry := range entries {
		if entry.IsDir() {
			coll = append(coll, entry.Name())
		}
	}
	return coll, nil
}

// ParseNamedRequest reads <dir>/<namespace>/http, locates the request block
// whose @name matches namedRequest, substitutes {KEY} placeholders using env,
// and parses the resulting bytes into an *http.Request. The env files are not
// touched — the caller is expected to have resolved env already (typically via
// CollectionContext).
func ParseNamedRequest(dir, namespace, namedRequest string, env map[string]any) (*http.Request, error) {
	collection := getFilePath(dir, namespace, "http")

	collectionFile, err := os.Open(collection)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s", collection)
	}
	defer collectionFile.Close()

	scanner := bufio.NewScanner(collectionFile)
	var currentBlock []string
	var currentName string
	var foundBlock []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// "###" delimits request blocks. Evaluate the previous block first.
		if strings.HasPrefix(trimmed, "###") {
			if currentName == namedRequest && len(currentBlock) > 0 {
				foundBlock = currentBlock
				break
			}
			currentBlock = nil
			currentName = ""
			continue
		}

		if isComment(trimmed) {
			if name, ok := extractName(trimmed); ok {
				currentName = name
			}
			continue
		}

		// Add functional HTTP lines (request line, headers, body) to the current block.
		if line != "" || len(currentBlock) > 0 {
			currentBlock = append(currentBlock, line)
		}
	}

	// Catch the last block if the file didn't end with a "###" delimiter.
	if currentName == namedRequest && len(currentBlock) > 0 && len(foundBlock) == 0 {
		foundBlock = currentBlock
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	if len(foundBlock) == 0 {
		return nil, fmt.Errorf("request with name %q not found", namedRequest)
	}

	rawHTTP := strings.Join(foundBlock, "\r\n")

	// Substitute {KEY} placeholders using the caller-supplied env.
	for key, value := range env {
		rawHTTP = strings.ReplaceAll(rawHTTP, fmt.Sprintf("{%s}", key), value.(string))
	}

	// Fix missing HTTP version on the request line.
	firstLineEnd := strings.Index(rawHTTP, "\r\n")
	if firstLineEnd == -1 {
		firstLineEnd = len(rawHTTP)
	}
	firstLine := rawHTTP[:firstLineEnd]
	if !strings.Contains(firstLine, " HTTP/") {
		rawHTTP = strings.Replace(rawHTTP, firstLine, firstLine+" HTTP/1.1", 1)
	}

	// http.ReadRequest leaves req.Body empty unless Content-Length (or chunked
	// encoding) is set, so a POST with a body but no length header silently
	// goes out with zero bytes. Inject Content-Length from the body we parsed.
	if sepIdx := strings.Index(rawHTTP, "\r\n\r\n"); sepIdx != -1 {
		headers := rawHTTP[:sepIdx]
		body := strings.TrimRight(rawHTTP[sepIdx+4:], "\r\n")
		if body != "" && !hasContentLengthHeader(headers) {
			headers += "\r\nContent-Length: " + strconv.Itoa(len(body))
		}
		rawHTTP = headers + "\r\n\r\n" + body
	}

	// Ensure proper trailing double CRLF termination.
	if !strings.HasSuffix(rawHTTP, "\r\n\r\n") {
		if strings.HasSuffix(rawHTTP, "\r\n") {
			rawHTTP += "\r\n"
		} else {
			rawHTTP += "\r\n\r\n"
		}
	}

	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader([]byte(rawHTTP))))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTTP request text: %w", err)
	}

	// ReadRequest parses the path but leaves URL.Host empty; rebuild a complete URL.
	if req.URL.Host == "" && req.Host != "" {
		req.URL.Host = req.Host
		if req.TLS == nil {
			req.URL.Scheme = "http"
		} else {
			req.URL.Scheme = "https"
		}
	}

	return req, nil
}

// GetEnvVariables returns globalenv merged with <namespace>/env (namespace
// values win on key conflicts). Used by the env subcommand for display.
// The collection command path uses CollectionContext instead.
func GetEnvVariables(dir, namespace string) map[string]any {
	localvars, _ := config.Load(getFilePath(dir, namespace, "env"))
	return Merge(GetGlobalEnvVariables(), localvars)
}

func GetGlobalEnvVariables() map[string]any {
	globalvars, _ := config.Load(config.GetGlobalEnvFile())
	return globalvars
}

func hasContentLengthHeader(headers string) bool {
	for _, line := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "content-length:") {
			return true
		}
	}
	return false
}

func GetRequestsForNamespace(dir, namespace string) ([]string, error) {
	collection := getFilePath(dir, namespace, "http")

	collectionFile, err := os.Open(collection)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s", collection)
	}
	defer collectionFile.Close()

	scanner := bufio.NewScanner(collectionFile)
	var requests []string

	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if !isComment(trimmed) {
			continue
		}
		if name, ok := extractName(trimmed); ok && name != "" {
			requests = append(requests, name)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return requests, nil
}

// isComment reports whether a trimmed line is a "#" or "//" style comment.
func isComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//")
}

// extractName returns the value of a "@name foo" annotation in a comment line.
// The "###" separator is not an @name carrier.
func extractName(trimmed string) (string, bool) {
	cleanComment := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "#"), "//"))
	if !strings.HasPrefix(cleanComment, "@name") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(cleanComment, "@name")), true
}

func parseVariables(variables []string) map[string]string {
	varmap := map[string]string{}
	for _, v := range variables {
		parts := strings.Split(v, "=")
		varmap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return varmap
}
