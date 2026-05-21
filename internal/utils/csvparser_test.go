package utils_test

import (
	"testing"

	"github.com/parikhrahil/httpgo/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCsvParserInit(t *testing.T) {
	tests := []struct {
		name string
		filepath string
		wantError bool
	}{
		{"success", "./testdata/test.csv", false},
		{"no File", "./testdata/nofile.csv", true},
		{"empty File", "./testdata/empty.csv", true},
		{"header only", "./testdata/header_only.csv", true},
		{"malformed CSV", "./testdata/malformed.csv", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := utils.NewCsvParser(tt.filepath)
			if !tt.wantError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParseCsv(t *testing.T) {
	row1 := map[string]string{"h1": "v11", "h2": "v12", "h3": "v13", "h4": "v14"}
	row2 := map[string]string{"h1": "v21", "h2": "v22", "h3": "v23", "h4": "v24"}
	row3 := map[string]string{"h1": "v31", "h2": "v32", "h3": "v33", "h4": "v34"}

	tests := []struct {
		name      string
		start     int
		end       int
		wantData  []map[string]string
		wantError bool
	}{
		{"valid slice", 1, 3, []map[string]string{row2, row3}, false},
		{"end > total rows", 0, 10, []map[string]string{row1, row2, row3}, false},
		{"start  > End", 3, 2, nil, true},
		{"start > total rows", 4, 5, nil, true},
	}

	filepath := "./testdata/test.csv"
	parser, err := utils.NewCsvParser(filepath)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := parser.ParseLines(tt.start, tt.end)
			if !tt.wantError {
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, data)
			} else {
				require.Error(t, err)
			}
		})
	}
}