package workerpool

import (
	"errors"
	"sync"
	"time"
)

type JobResult struct {
	Code     int
	Message  string
	ExecTime time.Duration
}

type Job interface {
	Run() *JobResult
}

type worker struct {
	id      int
	wg      *sync.WaitGroup
	jobs    <-chan Job
	results chan<- *JobResult
}

func (w *worker) handle() {
	defer w.wg.Done()
	for job := range w.jobs {
		w.results <- job.Run()
	}
}

type Pool struct {
	size        int
	wg          *sync.WaitGroup
	JobsChan    chan Job
	ResultsChan chan *JobResult
}

func (p *Pool) Join() {
	p.wg.Wait()
	close(p.ResultsChan)
}

func NewPool(size int) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("Pool size cannot be negative")
	}

	wg := &sync.WaitGroup{}
	jobsChan := make(chan Job, size)
	resultsChan := make(chan *JobResult, size*3)

	for i := 0; i < size; i++ {
		worker := &worker{
			i, wg, jobsChan, resultsChan,
		}
		wg.Add(1)
		go worker.handle()
	}

	return &Pool{size, wg, jobsChan, resultsChan}, nil
}
