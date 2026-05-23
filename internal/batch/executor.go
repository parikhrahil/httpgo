package batch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	myhttp "github.com/parikhrahil/httpgo/internal/http"
	"github.com/parikhrahil/httpgo/internal/utils"
)

func fileReader(fp string) (readerFunc, error) {
	filetype := filepath.Ext(fp)

	switch filetype {
	case ".csv":
		return utils.NewCsvReader, nil
	case ".json":
		return utils.NewJsonReader, nil
	default:
		return nil, fmt.Errorf("input file type %s is not supported", filetype)
	}
}

func (b *BatchContext) Execute(c *utils.CollectionContext) error {
	readFunc, err := fileReader(b.inFile)
	if err != nil {
		return err
	}

	file, err := os.Open(b.inFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fileReader := readFunc(file)

	numWorkers := b.getConcurrency()
	const maxBufferSize = 100

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rowChan, errChan := fileReader.Read(ctx, maxBufferSize)

	resultChan := make(chan *result, maxBufferSize)

	var errWg sync.WaitGroup
	errWg.Add(1)

	errWg.Go(func() {
		defer errWg.Done()
		// This loop naturally terminates when Read closes errChan
		for e := range errChan {
			b.printError(e)
		}
	})

	var wg sync.WaitGroup
	var outputWg sync.WaitGroup

	outputWg.Add(1)
	outputWg.Go(func() {
		defer outputWg.Done()
		for r := range resultChan {
			b.printToConsole(r)
			b.writeOutput(r)
		}
	})

	// distribute work items to workers
	for range numWorkers {
		wg.Add(1)
		wg.Go(func() {
			defer wg.Done()
			for item := range rowChan {
				output, err := b.executeItem(item.Index, c, item.Data)
				if err != nil {
					b.printError(err)
					cancel()
				}
				resultChan <- output
			}
		})
	}

	wg.Wait() // Main thread blocked until workers are done.

	close(resultChan) // since jobs are done, no new res coming, signal og.
	outputWg.Wait()   // Main thread blocked until og is done.
	return nil
}

func (b *BatchContext) executeItem(index int, c *utils.CollectionContext, data map[string]any) (*result, error) {
	env := utils.Merge(c.Env(), data)
	req, err := utils.ParseNamedRequest(c.Dir, c.Namespace, b.request, env)
	if err != nil {
		return nil, err
	}

	if b.dryRun {
		return &result{req: req, index: index}, nil
	}

	start := time.Now()
	res, body, err := myhttp.ExecuteHTTPRequest(req, b.timeout)
	latency := time.Since(start)

	if err != nil {
		return nil, err
	}
	return &result{
		req:     req,
		res:     res,
		latency: latency,
		index:   index,
		body:    body,
	}, nil
}
