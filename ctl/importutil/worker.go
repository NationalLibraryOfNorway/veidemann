/*
 * Copyright 2021 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package importutil

import (
	"sync"
)

// Payload is an interface for the payload of a job in a work queue
type Payload interface {
	any
}

// Job represents a piece of work in the work queue
type Job[P Payload] struct {
	*State
	Val P
}

// Executor is a work queue that executes jobs in concurrent workers.
type Executor[P Payload] struct {
	Queue chan Job[P]
	stats chan error
	done  chan struct{}
	wg    sync.WaitGroup

	// Keep track of the number of jobs completed and the number of jobs that failed.
	count   int
	success int
	failed  int
}

// NewExecutor creates a work queue with nrOfWorkers workers.
//
// do is the function that will be called for each job.
// onError is the function that will be called for each job that fails.
//
// To close the work queue, call Wait() after all jobs have been queued.
// Writing to the Queue channel after Wait() has been called will panic.
func NewExecutor[P Payload](nrOfWorkers int, do func(P) error, onError func(Job[P])) *Executor[P] {
	e := &Executor[P]{
		Queue: make(chan Job[P], nrOfWorkers),
		done:  make(chan struct{}),
		stats: make(chan error, nrOfWorkers),
	}

	// start error handler
	go func() {
		defer close(e.done)
		for err := range e.stats {
			e.count++
			if err == nil {
				e.success++
			} else {
				e.failed++
			}
		}
	}()

	// start workers
	for i := 0; i < nrOfWorkers; i++ {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			for job := range e.Queue {
				err := do(job.Val)
				e.stats <- err
				if err != nil {
					job.err = err
					onError(job)
				}
			}
		}()
	}

	return e
}

// Wait waits for all jobs to complete.
// It returns the number of jobs completed, the number of jobs that succeeded and the number of jobs that failed.
func (e *Executor[P]) Wait() (int, int, int) {
	// stop workers
	close(e.Queue)
	// wait for workers to finish
	e.wg.Wait()
	// stop counting
	close(e.stats)
	// wait for counting to complete
	<-e.done
	// return stats
	return e.count, e.success, e.failed
}
