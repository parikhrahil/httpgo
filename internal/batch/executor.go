package batch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	const maxBufferSize = 100

	reader, err := fileReader(b.inFile)
	if err != nil {
		return err
	}

	file, err := os.Open(b.inFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fileReader := reader(file)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataStream, readErrorStream := fileReader.Read(ctx, maxBufferSize)
	results, errs := b.fanoutExecution(ctx, c, dataStream)
	resultStream, errStream := b.fanInResult(ctx, results, errs)

	var outwg sync.WaitGroup
	outwg.Add(3)

	go func() {
		defer outwg.Done()
		for err := range readErrorStream {
			b.printError(err)
		}
	}()

	go func() {
		defer outwg.Done()
		for res := range resultStream {
			b.printToConsole(res)
			b.writeOutput(res)
		}
	}()

	go func() {
		defer outwg.Done()
		for err := range errStream {
			b.printError(err)
		}
	}()

	outwg.Wait()
	return nil
}

func (b *BatchContext) fanInResult(ctx context.Context, results []<-chan *result, errs []<-chan error) (<-chan *result, <-chan error) {
	var wg sync.WaitGroup
	var eg sync.WaitGroup
	res := make(chan *result)
	err := make(chan error)

	transferRes := func(ctx context.Context, data <-chan *result) {
		defer wg.Done()
		for d := range data {
			select {
			case <-ctx.Done():
				return
			case res <- d:
			}
		}
	}

	transferErr := func(ctx context.Context, errs <-chan error) {
		defer eg.Done()
		for e := range errs {
			select {
			case <-ctx.Done():
				return
			case err <- e:
			}
		}
	}

	for _, r := range results {
		wg.Add(1)
		go transferRes(ctx, r)
	}

	for _, e := range errs {
		eg.Add(1)
		go transferErr(ctx, e)
	}

	go func() {
		wg.Wait()
		eg.Wait()
		close(res)
		close(err)
	}()

	return res, err
}

func (b *BatchContext) fanoutExecution(ctx context.Context, collCtx *utils.CollectionContext, data <-chan utils.DataItem) ([]<-chan *result, []<-chan error) {
	concurrency := b.getConcurrency()
	results := make([]<-chan *result, concurrency)
	errs := make([]<-chan error, concurrency)

	execute := func(ctx context.Context, data <-chan utils.DataItem) (<-chan *result, <-chan error) {
		reschan := make(chan *result)
		errchan := make(chan error)

		go func() {
			defer close(reschan)
			defer close(errchan)

			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-data:
					if !ok {
						return
					}
					res, err := b.executeItem(item.Index, collCtx, item.Data)
					if res != nil {
						reschan <- res
					}
					if err != nil {
						errchan <- err
					}
				}
			}
		}()

		return reschan, errchan
	}

	for i := range concurrency {
		results[i], errs[i] = execute(ctx, data)
	}

	return results, errs
}

func (b *BatchContext) executeItem(index int, c *utils.CollectionContext, data map[string]any) (*result, error) {
	env := utils.Merge(c.Env(), data)
	req, err := utils.ParseNamedRequest(c.Dir, c.Namespace, b.request, env)
	if err != nil {
		errReqFailed := errors.New("[" + strconv.Itoa(index) + "] " + err.Error())
		return nil, errReqFailed
	}

	if b.dryRun {
		return &result{req: req, index: index}, nil
	}

	start := time.Now()
	res, body, err := myhttp.ExecuteHTTPRequest(req, b.timeout)
	latency := time.Since(start)

	if err != nil {
		errRespFailed := errors.New("[" + strconv.Itoa(index) + "] " + err.Error())
		return nil, errRespFailed
	}
	return &result{
		req:     req,
		res:     res,
		latency: latency,
		index:   index,
		body:    body,
	}, nil
}
