package workflow

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

type FutureStage int

const (
	NOT_STARTED FutureStage = iota
	SUBMITTED
	RUNNING
	ABORTED
	TIMEDOUT
	TARGET_INVOKED
	COMPLETED
)

type Future struct {
	targetFunc   interface{}
	paramsPassed []interface{}
	funcReturned []interface{}
	voidReturn   bool
	rC           chan interface{} //response channel
	aC           chan error       //abort channel
	abortOnce    sync.Once
	runOnce      sync.Once
	stage        FutureStage
	es           *ExecutorService
}

func newFuture() *Future {

	f := &Future{}
	f.rC = make(chan interface{})
	f.stage = NOT_STARTED
	f.aC = make(chan error, 1)
	f.es = default_es

	return f
}

func (f *Future) SetExecutor(customExecutor *ExecutorService) *Future {
	f.es = customExecutor
	return f
}

// Call to Get() blocks until the target function invocation is completed.
// Return from this method indicates successful execution of target function or time-out or user aborted/cancelled this future.
// error is returned in case of timeouts or aborted.
// The return values of the target function is returned as array of interface{}.
func (f *Future) Get(timeout time.Duration) ([]interface{}, error) {

	if f.stage == ABORTED || f.stage == TIMEDOUT || f.stage == COMPLETED {
		switch f.stage {
		case ABORTED:
			return nil, fmt.Errorf("ABORTED")
		case TIMEDOUT:
			return nil, fmt.Errorf("TIMEDOUT")
		case COMPLETED:
			return nil, fmt.Errorf("COMPLETED ALREADY")
		}
	}

	if timeout > 0 {
		start := time.Now()
		tmr := time.NewTimer(timeout)
		select {
		case <-f.aC:
			fmt.Errorf("Aborted target (%+v)", reflect.TypeOf(f.targetFunc))
			f.funcReturned = nil
			return nil, fmt.Errorf("aborted")
		case <-tmr.C:
			end := time.Since(start).Milliseconds()
			fmt.Errorf("Timeout (%d) got triggered hence aborting target (%+v)", end, reflect.TypeOf(f.targetFunc))
			f.abortOnce.Do(f.abort)
			f.funcReturned = nil
			tmr.Stop()
			return nil, fmt.Errorf("timedout")
		case <-f.rC:
			tmr.Stop()
			f.stage = COMPLETED
			return f.funcReturned, nil
		}
	} else {
		select {
		case <-f.aC:
			fmt.Errorf("Aborted target (%+v)", reflect.TypeOf(f.targetFunc))
			f.funcReturned = nil
			return nil, fmt.Errorf("aborted")
		case <-f.rC:
			f.stage = COMPLETED
			return f.funcReturned, nil
		}
	}

}

// Cancel's the referred future, sends a signal to abort the call of target function.
// This method returns immediately.
// Call to Cancel() does not mean the target function is aborted if it is running already.
// Target function will not be called if it is not triggered yet when Cancel() is called.
// Any call to *Future.Get() after Cancel() is invoked will return an error indicating aborted
func (f *Future) Cancel() bool {
	if f.stage == COMPLETED {
		return false
	}
	f.abortOnce.Do(f.abort)
	f.stage = ABORTED
	return true
}

func (f *Future) abort() {
	fmt.Errorf("Cancelling target (%+v)", reflect.TypeOf(f.targetFunc))
	f.aC <- fmt.Errorf("aborted")
}

func (f *Future) Stage() FutureStage {
	return f.stage
}

func (f *Future) callTarget() []reflect.Value {
	targetType := reflect.TypeOf(f.targetFunc)
	valueOf := reflect.ValueOf(f.targetFunc)
	methodParams := make([]reflect.Value, 0, targetType.NumIn())
	for i, p := range f.paramsPassed {
		paramValue := reflect.ValueOf(p)
		if paramValue.Kind() == reflect.Invalid {
			paramValue = reflect.Zero(targetType.In(i))
		}
		methodParams = append(methodParams, paramValue)
	}

	fmt.Printf("Calling Target %s  no. of args = %d, no. of args passed = %d\n", valueOf.String(), targetType.NumIn(), len(methodParams))
	return valueOf.Call(methodParams)
}

func (f *Future) executeTarget() {
	defer close(f.rC)

	if len(f.aC) != 0 {
		return
	}
	f.stage = RUNNING
	returned := f.callTarget()
	f.stage = TARGET_INVOKED
	for _, r := range returned {
		f.funcReturned = append(f.funcReturned, r.Interface())
	}

}

// This has to be called to trigger the execution of the target function by a GOROUTINE.
// The target function will not be executed unless this method is invoked.
func (f *Future) Execute() *Future {
	if f.stage != NOT_STARTED {
		return f
	}
	f.runOnce.Do(func() {
		f.es.Submit(f.executeTarget)
	})
	f.stage = SUBMITTED
	return f
}

// This method creates a Future that represents the async execution of the target function.
// Execute() should be called on the returned future to trigger the execution of the target function.
func RunAsync(targetFunc interface{}, args ...interface{}) (*Future, error) {
	targetType := reflect.TypeOf(targetFunc)
	switch targetType.Kind() {
	case reflect.Func:
		if targetType.NumIn() != len(args) && !targetType.IsVariadic() {
			fmt.Errorf("mismatch in number of arguments, expected = %d, passed = %d", targetType.NumIn(), len(args))
			return nil, fmt.Errorf("mismatch in number of arguments, expected = %d, passed = %d", targetType.NumIn(), len(args))
		}

		future := newFuture()
		future.targetFunc = targetFunc
		future.paramsPassed = args
		future.voidReturn = targetType.NumOut() == 0
		return future, nil
	default:
		return nil, fmt.Errorf("%s un-supported type", targetType.Kind())
	}
}
