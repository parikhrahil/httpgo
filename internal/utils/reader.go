package utils

import "context"

type FileReader interface {
	Read(ctx context.Context, bufferSize int) (<-chan DataItem, <-chan error)
}

type DataItem struct {
	Index int
	Data  map[string]any
}
