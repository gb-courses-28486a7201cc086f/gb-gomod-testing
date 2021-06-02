// Package implements simple pool of workers.
// Pool has two channels: input and output.
//
// Input channel is a queue of Jobs (single unit of work).
// Workers takes Job, executes Job.Run() method and
// puts result object into Pool's output channel.
//
// Also Pool provides Pool.Join() method to blck caller
// until all workers finished
package workerpool

import (
	"errors"
	"sync"
	"time"
)

// JobResult object stores result of completed Job
type JobResult struct {
	Code     int
	Message  string
	ExecTime time.Duration
}

// Job is interface which used by workers to
// make job
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

// Poll is manager of set of workers.
// Provides two channels:
//
// * JobsChan - input queue of Jobs, size = count of workers.
// So, you can put one Job per worker until blocked
//
// * ResultsChan - output queue
type Pool struct {
	size        int
	wg          *sync.WaitGroup
	JobsChan    chan Job
	ResultsChan chan *JobResult
}

// Join method waits until all workers finished
// and close result channel to notify caller what all Jobs done
func (p *Pool) Join() {
	p.wg.Wait()
	close(p.ResultsChan)
}

// NewPool is a fabric method which initiates all
// data structures underhood of Pool.
// Takes required count of workers as argument.
//
// Creates channels:
//
// * JobsChan (input) with size = count of workers.
// So, you can put one Job per worker until 'producer' blocked
//
// * ResultsChan (output) with size = count of workers * 3.
// So, you each worker can return a few results until pool
// will be blocked
//
// Be careful! If caller does not read results - deadlock will occurred
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
