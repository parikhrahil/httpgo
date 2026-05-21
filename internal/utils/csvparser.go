package utils

import (
	"encoding/csv"
	"fmt"
	"os"
)

// CsvParser holds a parsed CSV file's header keys and remaining data rows.
type CsvParser struct {
	keys []string
	rows [][]string
}

// NewCsvParser reads the CSV file at fp, treats the first record as the header,
// and returns a parser positioned over the remaining data rows. It returns an
// error if the file cannot be opened, is not valid CSV, or contains no data
// rows beyond the header.
func NewCsvParser(fp string) (*CsvParser, error) {
	file, err := os.Open(fp)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	csvReader := csv.NewReader(file)

	rows, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("no data found in %s", fp)
	}

	keys := make([]string, len(rows[0]))
	copy(keys, rows[0])

	return &CsvParser{
		keys: keys,
		rows: rows[1:],
	}, nil
}

// ParseLines returns the rows in the half-open range [start, end) as
// header-keyed maps. end is clamped to the number of available data rows;
// start must be strictly less than end and within bounds.
func (c *CsvParser) ParseLines(start, end int) ([]map[string]string, error) {
	if start >= end {
		return nil, fmt.Errorf("start index should be less than end index")
	}

	if start >= len(c.rows) {
		return nil, fmt.Errorf("start index %d out of bounds (available rows: %d)", start, len(c.rows))
	}

	if end > len(c.rows) {
		end = len(c.rows)
	}

	result := []map[string]string{}

	for i, row := range c.rows[start : end] {
		result = append(result, map[string]string{})
		for j, val := range row {
			key := c.keys[j]
			result[i][key] = val
		}
	}

	return result, nil
}
