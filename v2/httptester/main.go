// Package main implements cli-tool to send parallel
// HTTP requests to specified URL and calculate average response time.
//
// Run tool with -h parameter to see all cli flags
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gb-courses-28486a7201cc086f/gb-gomod-testing/v2/httptester/workerpool"
)

var (
	workers     int
	requests    int
	timeout     time.Duration
	url         string
	method      string
	payload     string
	httpTimeOut time.Duration

	errEmptyURL = errors.New("URL flag \"-url\" should be provided")
)

func setUp() (*Config, error) {
	flag.IntVar(&workers, "w", 100, "Parallel workers which perform requests")
	flag.IntVar(&requests, "c", 0, "Total count of requests to send")
	flag.DurationVar(&timeout, "t", time.Second*0, "Time limit to perform test. 0 means no time limit")
	flag.StringVar(&url, "url", "", "URL to test")
	flag.StringVar(&method, "method", "GET", "HTTP method for test requests")
	flag.StringVar(&payload, "data", "", "Payload for test requests")
	flag.DurationVar(&httpTimeOut, "reqtimeout", time.Second*5, "Timeout for HTTP client")
	flag.Parse()

	if url == "" {
		return nil, errEmptyURL
	}

	return &Config{
		Workers:     workers,
		Requests:    requests,
		Timeout:     timeout,
		URL:         url,
		Method:      method,
		Payload:     []byte(payload),
		HTTPTimeOut: httpTimeOut,
	}, nil
}

func produceJobs(config *Config, jobsChan chan<- workerpool.Job) {
	client := &http.Client{Timeout: config.HTTPTimeOut}

	// timer to stop on duration
	var timerChan <-chan time.Time
	if config.Timeout.Milliseconds() > 0 {
		timerChan = time.NewTimer(config.Timeout).C
	}

	// produce jobs
	jobsCount := 0
	running := true
	for running {
		jobsCount++
		job := &TestURLJob{
			ID:      jobsCount,
			Client:  client,
			URL:     config.URL,
			Method:  config.Method,
			Payload: bytes.NewReader([]byte{}),
		}

		select {
		case jobsChan <- job:
			// send new job and
			// check if jobs counter exceed,
			// config.Requests==0 means infinite test
			if config.Requests != 0 && jobsCount >= config.Requests {
				running = false
			}
		case <-timerChan:
			// stop on timeout
			running = false
		}
	}
}

func main() {
	config, err := setUp()
	if err != nil {
		fmt.Println(err)
		os.Exit(255)
	}
	fmt.Printf("Starting test for %s %s\n\n", config.Method, config.URL)

	pool, err := workerpool.NewPool(config.Workers)
	if err != nil {
		fmt.Println(err)
		os.Exit(255)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	// collect anf print results
	go JobReporter(wg, pool.ResultsChan)

	// send jobs until timeout or count exceed
	produceJobs(config, pool.JobsChan)

	// shutdown gracefully
	close(pool.JobsChan)
	pool.Join()
	wg.Wait()
}
