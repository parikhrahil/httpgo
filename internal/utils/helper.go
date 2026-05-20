package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/parikhrahil/httpgo/internal/config"
)

type PrintProps struct {
	Raw     bool
	Pretty  bool
	Headers bool
}

func PrettyJson(body []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		fmt.Println("Error:", err)
		return string(body)
	}
	return prettyJSON.String()
}

func getFilePath(elems ...string) string {
	return path.Join(elems...)
}

func WriteToFile(filepath string, body []byte) {
	if len(body) == 0 {
		fmt.Printf("No body to write!\n")
		return
	}

	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Could not open or create the file: %q\n", filepath)
		return
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, string(body)); err != nil {
		fmt.Printf("Could not write the response to file: %q\n", filepath)
		return
	}
	fmt.Printf("Response saved to %q\n", filepath)
}

// PrintRequest dumps the outgoing request's method, URL, headers, and body
// (or parsed form values when the body is form-encoded). It reads req.Body
// and restores it so the request can still be sent afterwards.
func PrintRequest(req *http.Request) {
	fmt.Printf("Method: %s\nURL: %s\n", req.Method, req.URL.String())

	if len(req.Header) > 0 {
		fmt.Println("\nRequest Headers:")
		for key, values := range req.Header {
			for _, v := range values {
				fmt.Printf("  %s: %s\n", key, v)
			}
		}
	}

	if req.Body == nil {
		return
	}
	bodyBytes, err := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if err != nil || len(bodyBytes) == 0 {
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if values, err := url.ParseQuery(string(bodyBytes)); err == nil && len(values) > 0 {
			fmt.Println("\nRequest Form:")
			for key, vals := range values {
				for _, v := range vals {
					fmt.Printf("  %s: %s\n", key, v)
				}
			}
			return
		}
	}

	fmt.Println("\nRequest Body:")
	if strings.Contains(contentType, "application/json") {
		fmt.Println(PrettyJson(bodyBytes))
	} else {
		fmt.Println(string(bodyBytes))
	}
}

func PrintToConsole(props *PrintProps, res *http.Response, body []byte) {
	if !props.Raw {
		fmt.Printf("\nResponse Status: %s\n", res.Status)
	}

	if !props.Raw && props.Headers {
		fmt.Println("Response Headers:")
		for key, values := range res.Header {
			fmt.Printf("  %s: %s\n", key, values[0])
		}
	}

	if props.Raw {
		fmt.Println(string(body))
		return
	}

	fmt.Println("\nResponse Body:")
	if len(body) == 0 {
		fmt.Println("[Empty Body]")
		return
	}
	if props.Pretty {
		fmt.Println(PrettyJson(body))
	} else {
		fmt.Println(string(body))
	}
}

func IsValidNamespace(ns string) bool {
	wd := config.GetWorkingDirectory()
	namespaces, err := GetValidCollections(wd)
	if err != nil {
		panic("Error loading collections from " + wd)
	}
	return slices.Contains(namespaces, ns)
}
