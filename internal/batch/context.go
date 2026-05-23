package batch

import (
	"io"
	"net/http"
	"time"

	"github.com/parikhrahil/httpgo/internal/utils"
)

type BatchContext struct {
	inFile        string
	request       string
	concurrency   uint
	timeout       time.Duration
	dryRun        bool
	outFile       string
	includeBody   bool
	headerWritten bool
}

type BatchOpts struct {
	Request     string
	Filepath    string
	Concurrency uint
	Timeout     time.Duration
	DryRun      bool
	OutFile     string
	IncludeBody bool
}

type result struct {
	req     *http.Request
	res     *http.Response
	body    []byte
	latency time.Duration
	index   int
}

type readerFunc func(input io.Reader) utils.FileReader

func NewContext(opts *BatchOpts) (*BatchContext, error) {
	return &BatchContext{
		inFile:      opts.Filepath,
		request:     opts.Request,
		concurrency: opts.Concurrency,
		timeout:     opts.Timeout,
		dryRun:      opts.DryRun,
		outFile:     opts.OutFile,
		includeBody: opts.IncludeBody,
	}, nil
}

func (b *BatchContext) getConcurrency() int {
	n := b.concurrency
	if n == 0 {
		n = 1
	}
	return int(n)
}
