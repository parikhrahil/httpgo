package utils

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

type JsonReader struct {
	scanner *bufio.Scanner
}

func NewJsonReader(input io.Reader) FileReader {
	return &JsonReader{
		scanner: bufio.NewScanner(input),
	}
}

func (jsonReader *JsonReader) Read(ctx context.Context, bufferSize int) (<-chan DataItem, <-chan error) {
	outchan := make(chan DataItem, bufferSize)
	errchan := make(chan error, bufferSize)

	go func() {
		var index int

		for jsonReader.scanner.Scan() {
			index++
			var data map[string]any
			if err := json.Unmarshal(jsonReader.scanner.Bytes(), &data); err != nil {
				errMalformedjson := errors.New("[" + strconv.Itoa(index) + "] malformed row")
				if !SendErr(ctx, errchan, errMalformedjson) {
					return
				}
				continue
			}

			item := DataItem{Index: index, Data: data}

			select {
			case outchan <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	return outchan, errchan
}
