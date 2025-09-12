package completion

import (
	"time"
)

// deferredRunner implements a runner that is executed
type deferredRunner[T any] struct {
	delay      time.Duration
	timer      *time.Timer
	isRunning  bool
	resultChan chan []T
}

func newRunner[T any](delay time.Duration) *deferredRunner[T] {
	return &deferredRunner[T]{
		delay:      delay,
		resultChan: make(chan []T, 1),
	}
}

func (r *deferredRunner[T]) SetDelay(delay time.Duration) {
	r.delay = delay
}

func (r *deferredRunner[T]) Run(deferredFunc func() []T) {
	if r.delay == 0 {
		r.Async(deferredFunc)
	} else {
		r.deferRun(deferredFunc)
	}
}

func (r *deferredRunner[T]) deferRun(deferredFunc func() []T) {
	if !r.IsRunning() {
		r.start(deferredFunc)
	}

	r.timer.Reset(r.delay)
}

func (r *deferredRunner[T]) Async(deferredFunc func() []T) {
	if r.isRunning {
		r.timer.Stop()
	}

	go func() {
		r.resultChan <- deferredFunc()
	}()
}

func (r *deferredRunner[T]) start(deferredFunc func() []T) {
	if r.timer == nil {
		r.timer = time.NewTimer(r.delay)
	} else {
		r.timer.Reset(r.delay)
	}

	if r.isRunning {
		return
	}

	go func() {
		r.isRunning = true
		defer func() { r.isRunning = false }()

		<-r.timer.C
		r.resultChan <- deferredFunc()
		logger.Debug("Running deferred function...")

	}()

}

func (r *deferredRunner[T]) IsRunning() bool {
	return r.isRunning
}

func (r *deferredRunner[T]) ResultChan() <-chan []T {
	return r.resultChan
}
