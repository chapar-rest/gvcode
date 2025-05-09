package completion

import (
	"time"
)

// deferredRunner implements a runner that is executed  should be bound to a dedicated editor.
type deferredRunner[T any] struct {
	delay      time.Duration
	timer      *time.Timer
	isRunning  bool
	resultChan chan []T
}

func newRunner[T any](delay time.Duration) *deferredRunner[T] {
	if delay == time.Duration(0) {
		delay = time.Millisecond * 50
	}

	return &deferredRunner[T]{
		delay:      delay,
		resultChan: make(chan []T, 1),
	}
}

func (r *deferredRunner[T]) Defer(deferredFunc func() []T) {
	if !r.IsRunning() {
		r.start(deferredFunc)
	}

	r.timer.Reset(r.delay)
}

func (r *deferredRunner[T]) Async(deferredFunc func() []T) {
	if r.delay != 0 {
		return
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
