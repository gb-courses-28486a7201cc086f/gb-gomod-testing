package workerpool

import (
	"fmt"
	"log"
	"time"
)

type JobMock int64

func (j JobMock) Run() *JobResult {
	return &JobResult{
		Code:     0,
		Message:  fmt.Sprintf("done job %d", j),
		ExecTime: 1 * time.Second,
	}
}

func Example() {
	// pool creation
	workers := 10
	pool, err := NewPool(workers)
	if err != nil {
		log.Println(err)
	}

	// job 'producing'
	for i := 0; i < workers; i++ {
		job := JobMock(i)
		pool.JobsChan <- job
	}

	// close pool
	pool.Join()

	// takes resuts
	for res := range pool.ResultsChan {
		log.Println(res)
	}
}
