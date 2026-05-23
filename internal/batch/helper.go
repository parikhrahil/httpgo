package batch

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/parikhrahil/httpgo/internal/utils"
)

func (b *BatchContext) printToConsole(res *result) {
	if b.dryRun {
		fmt.Printf("\n%d\n", res.index)
		utils.PrintRequest(res.req)
	} else {
		fmt.Printf("[%d] %d %6dms %6s %s\n", res.index, res.res.StatusCode, res.latency.Milliseconds(), res.req.Method, res.req.URL.String())
	}
}

func (b *BatchContext) printError(err error) {
	fmt.Printf("ERROR %s\n", err.Error())
}

func (b *BatchContext) writeOutput(result *result) error {
	if b.outFile == "" {
		return nil
	}

	outFileType := filepath.Ext(b.outFile)
	switch outFileType {
	case ".csv":
		b.writeCSVHeader()
		if result != nil {
			return b.writeCSVToFile(result)
		}
	default:
		return fmt.Errorf("Unsupported output file type %s", outFileType)
	}
	return nil
}

func (b *BatchContext) writeCSVToFile(res *result) error {
	var content []byte
	_, err := b.marshalCSV(res, &content)
	if err != nil {
		return err
	}

	return b.writeToFile(content)
}

func (b *BatchContext) writeToFile(content []byte) error {
	file, err := os.OpenFile(b.outFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return err
	}
	return nil
}

func (b *BatchContext) marshalCSV(res *result, content *[]byte) (int, error) {
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(res.index))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%d", res.res.StatusCode))
	sb.WriteString(",")
	sb.WriteString(fmt.Sprintf("%d", res.latency.Milliseconds()))
	sb.WriteString(",")
	sb.WriteString(res.req.Method)
	sb.WriteString(",")
	sb.WriteString(res.req.URL.String())
	if b.includeBody {
		sb.WriteString(",")
		sb.WriteString(string(res.body))
	}
	sb.WriteString("\n")
	*content = []byte(sb.String())
	return len(*content), nil
}

func (b *BatchContext) writeCSVHeader() error {
	if b.headerWritten {
		return nil
	}
	file, err := os.OpenFile(b.outFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	header := "row,status,latencyMs,method,url"
	if b.includeBody {
		header = header + ",body"
	}
	header = header + "\n"
	if _, err := file.Write([]byte(header)); err != nil {
		return err
	}
	b.headerWritten = true
	return nil
}
