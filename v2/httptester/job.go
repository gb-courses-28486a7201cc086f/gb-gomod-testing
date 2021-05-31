package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gb-courses-28486a7201cc086f/gb-gomod-testing/v2/httptester/workerpool"
)

// Config contains parameters of test:
// - number of workers to run in parallel
// - max number of requests
// - max time of test
// - URL to send request
// - request method and payload
// - http-client timeout
type Config struct {
	Workers     int
	Requests    int
	Timeout     time.Duration
	URL         string
	Method      string
	Payload     []byte
	HTTPTimeOut time.Duration
}

// TestURLJob represents a single unit of work.
// TestURLJob is self-contained - all data for test stored inside
// Workers executes TestURLJob.Run() method to make test
type TestURLJob struct {
	ID      int
	Client  *http.Client
	URL     string
	Method  string
	Payload *bytes.Reader
}

// Run method executes test case using receiver attributes
func (tj *TestURLJob) Run() *workerpool.JobResult {
	req, err := http.NewRequest(tj.Method, tj.URL, tj.Payload)
	if err != nil {
		return &workerpool.JobResult{
			Code:     -1,
			Message:  err.Error(),
			ExecTime: 0,
		}
	}
	start := time.Now()
	resp, err := tj.Client.Do(req)
	execTime := time.Since(start)
	if err != nil {
		return &workerpool.JobResult{
			Code:     -1,
			Message:  err.Error(),
			ExecTime: execTime,
		}
	}
	defer resp.Body.Close()

	return &workerpool.JobResult{
		Code:     resp.StatusCode,
		Message:  "",
		ExecTime: execTime,
	}
}

// JobReport object aggregares results of test jobs
type JobReport struct {
	reqTotal            int
	reqSuccess          int
	reqServerErrors     map[int]int
	reqFailed           int
	reqSuccessTotalTime time.Duration
	startTime           time.Time
}

// Update method collect results from passed JobResult object
func (jr *JobReport) Update(data *workerpool.JobResult) {
	jr.reqTotal++
	// collect failed requests (client/network issues)
	if data.Code < 0 {
		jr.reqFailed++
		return
	}

	// collect success jobs count and exec time
	if data.Code >= 200 && data.Code < 500 {
		// non server errors
		jr.reqSuccess++
		jr.reqSuccessTotalTime += data.ExecTime
	} else {
		jr.reqServerErrors[data.Code] += 1
	}
}

// AvgTimeReport calculates average execution via already collected
// resulcts time and returns formatted message
func (jr *JobReport) AvgTimeReport() string {
	if jr.reqSuccess > 0 {
		avgSuccessTime := jr.reqSuccessTotalTime.Seconds() / float64(jr.reqSuccess)
		return fmt.Sprintf("%d requests: avg response time, sec: %.3f", jr.reqSuccess, avgSuccessTime)
	}
	return ""
}

// FinalReport calculates tolal requests, average time, RPC
// and creates formatted message
func (jr *JobReport) FinalReport() (msg string) {
	sentReq := float64(jr.reqTotal - jr.reqFailed)
	testExecTime := time.Since(jr.startTime).Seconds()
	rpsSuccess := float64(jr.reqSuccess) / testExecTime

	//runtime.Breakpoint()

	msg += fmt.Sprintf("Results:\nSent %.0f requests, %d success", sentReq, jr.reqSuccess)
	if len(jr.reqServerErrors) > 0 {
		msg += fmt.Sprintf("\nServer errors by code: %v\n", jr.reqServerErrors)
	}
	msg += fmt.Sprintf("\nRPS via success: %.000f\n", rpsSuccess)
	return msg
}

// JobReporter takes workers results from channel, aggregates it.
// Progress and final results are printed to console
func JobReporter(wg *sync.WaitGroup, resultsChan <-chan *workerpool.JobResult) {
	defer wg.Done()

	report := &JobReport{
		reqServerErrors: make(map[int]int),
		startTime:       time.Now(),
	}

	// setup ticker to print sub results
	ticker := time.NewTicker(time.Second)

	// cleanup
	defer func() {
		// all results done -> ticker can stop
		ticker.Stop()
		// print final result
		log.Println(report.AvgTimeReport())
		log.Println(report.FinalReport())
	}()

	for {
		select {
		case v, ok := <-resultsChan:
			// resultsChan closed => stop report
			if !ok {
				return
			}
			report.Update(v)
		case <-ticker.C:
			if msg := report.AvgTimeReport(); msg != "" {
				log.Println(msg)
			}
		}
	}
}
