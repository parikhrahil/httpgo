package utils

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
)

type CsvReader struct {
	reader *csv.Reader
}

func NewCsvReader(input io.Reader) FileReader {
	return &CsvReader{
		reader: csv.NewReader(input),
	}
}

func (csvReader *CsvReader) Read(ctx context.Context, bufferSize int) (<-chan DataItem, <-chan error) {
	outchan := make(chan DataItem, bufferSize)
	errchan := make(chan error, bufferSize)

	go func() {
		defer close(outchan)
		defer close(errchan)

		headers, err := csvReader.reader.Read()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				SendErr(ctx, errchan, err)
			}
			return
		}

		var index int
		for {
			index++
			record, err := csvReader.reader.Read()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				SendErr(ctx, errchan, err)
				continue
			}

			if len(record) != len(headers) {
				errMalformed := errors.New("[" + strconv.Itoa(index) + "] malformed row")
				if !SendErr(ctx, errchan, errMalformed) {
					return
				}
				continue // skip this row and proceed to next
			}

			data := getDataItem(index, headers, record)

			select {
			case outchan <- data:
			case <-ctx.Done():
				return
			}
		}
	}()
	return outchan, errchan
}

func getDataItem(index int, headers []string, record []string) DataItem {
	data := make(map[string]any, len(headers))

	for i := 0; i < len(headers); i++ {
		data[headers[i]] = record[i]
	}

	return DataItem{Index: index, Data: data}
}
