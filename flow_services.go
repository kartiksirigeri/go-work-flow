package workflow

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

type OpType int

const (
	CALL OpType = iota
	AND
	APPLY
	COMBINE
)

type step struct {
	targetFunc   interface{}
	paramsPassed []interface{}
	funcReturned []interface{}
	voidReturn   bool
	op           OpType
	future       *Future
}

type Flow struct {
	steps   []*step
	future  *Future
	runOnce sync.Once
	es      *ExecutorService
}

func (fl *Flow) SetExecutor(customExecutor *ExecutorService) *Flow {
	fl.es = customExecutor
	return fl
}

// This method creates a new Flow and adds a step that represents the async execution of the target function.
// Execute() should be called on the returned flow to trigger the execution of the target function.
func NewFlow(targetFunc interface{}, args ...interface{}) *Flow {
	targetType := reflect.TypeOf(targetFunc)
	switch targetType.Kind() {
	case reflect.Func:
		flow := &Flow{}
		step0 := &step{}
		step0.targetFunc = targetFunc
		step0.paramsPassed = args
		step0.voidReturn = targetType.NumOut() == 0
		step0.op = CALL
		flow.steps = append(flow.steps, step0)
		flow.es = default_es
		return flow
	default:
		fmt.Errorf("%s un-supported type", targetType.Kind())
		return nil
	}
}

// This method creates a new step in the Flow that represents the async execution of the target function.
// Step created using this method represents a target function is independent and can be triggered independently.
func (fl *Flow) AndCall(targetFunc interface{}, args ...interface{}) *Flow {
	targetType := reflect.TypeOf(targetFunc)
	switch targetType.Kind() {
	case reflect.Func:
		nxtStp := &step{}
		nxtStp.targetFunc = targetFunc
		nxtStp.paramsPassed = args
		nxtStp.voidReturn = targetType.NumOut() == 0
		nxtStp.op = AND
		fl.steps = append(fl.steps, nxtStp)
		return fl
	default:
		fmt.Errorf("%s un-supported type", targetType.Kind())
		return nil
	}
}

// This method creates a new step in the Flow that represents the async execution of the target function.
// Step created using this method represents a target function that combines the results of the previous steps.
// The argument of this target function should match the return types of the previous steps.
func (fl *Flow) ThenCombine(targetFunc interface{}) *Flow {
	targetType := reflect.TypeOf(targetFunc)
	switch targetType.Kind() {
	case reflect.Func:
		nxtStp := &step{}
		nxtStp.targetFunc = targetFunc
		nxtStp.voidReturn = targetType.NumOut() == 0
		nxtStp.op = COMBINE
		fl.steps = append(fl.steps, nxtStp)
		return fl
	default:
		fmt.Errorf("%s un-supported type", targetType.Kind())
		return nil
	}
}

// This method creates a new step in the Flow that represents the async execution of the target function.
// Step created using this method represents a target function, this step runs after the execution of the previous step.
// The argument of this target function should match the return types of the previous step.
func (fl *Flow) ThenApply(targetFunc interface{}) *Flow {
	targetType := reflect.TypeOf(targetFunc)
	switch targetType.Kind() {
	case reflect.Func:
		nxtStp := &step{}
		nxtStp.targetFunc = targetFunc
		nxtStp.voidReturn = targetType.NumOut() == 0
		nxtStp.op = APPLY
		fl.steps = append(fl.steps, nxtStp)
		return fl
	default:
		fmt.Errorf("%s un-supported type", targetType.Kind())
		return nil
	}
}

func (fl *Flow) runFlow() ([]interface{}, error) {

	for i := range fl.steps {
		currentStp := fl.steps[i]

		switch currentStp.op {
		case CALL, AND:
			if callFtr, err := RunAsync(currentStp.targetFunc, currentStp.paramsPassed...); err != nil {
				fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i].targetFunc, fl.steps[i].op, err.Error())
				return nil, fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i].targetFunc, fl.steps[i].op, err.Error())
			} else {
				currentStp.future = callFtr
				callFtr.SetExecutor(fl.es).Execute()
			}

		case APPLY:
			if stepOutput, err := fl.steps[i-1].future.Get(0); err != nil {
				fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i-1].targetFunc, fl.steps[i-1].op, err.Error())
				return nil, fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i-1].targetFunc, fl.steps[i-1].op, err.Error())
			} else {
				if applyFtr, err := RunAsync(currentStp.targetFunc, stepOutput...); err != nil {
					fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i].targetFunc, fl.steps[i].op, err.Error())
					return nil, fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i].targetFunc, fl.steps[i].op, err.Error())
				} else {
					currentStp.future = applyFtr
					applyFtr.SetExecutor(fl.es).Execute()
				}
			}
		case COMBINE:
			var allResponses []interface{}
			start := 0
			for p := 0; p < i; p++ {
				if fl.steps[p].op == APPLY || fl.steps[p].op == COMBINE {
					start = p
				}
			}
			for p := start; p < i; p++ {
				if pCallFtr, err := fl.steps[p].future.Get(0); err != nil {
					fmt.Errorf("%+v step (%d) failed with %s", fl.steps[p].targetFunc, fl.steps[p].op, err.Error())
					return nil, fmt.Errorf("%+v step (%d) failed with %s", fl.steps[p].targetFunc, fl.steps[p].op, err.Error())
				} else {
					allResponses = append(allResponses, pCallFtr...)
				}
			}
			if combinedFtr, err := RunAsync(currentStp.targetFunc, allResponses...); err != nil {
				fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i].targetFunc, fl.steps[i].op, err.Error())
				return nil, fmt.Errorf("%+v step (%d) failed with %s", fl.steps[i].targetFunc, fl.steps[i].op, err.Error())
			} else {
				currentStp.future = combinedFtr
				combinedFtr.SetExecutor(fl.es).Execute()
			}
		}

	}

	var (
		flowReturn []interface{}
		flowError  error = fmt.Errorf("flow aborted/timedout")
	)
	if fl.steps[len(fl.steps)-1].future != nil { //this is to check for aborts/timeout
		flowReturn, flowError = fl.steps[len(fl.steps)-1].future.Get(0)
	}
	fmt.Println("completed flow")
	return flowReturn, flowError
}

// This has to be called to trigger the execution of steps/pipeline represented by this flow.
// The target function(s) will not be executed unless this method is invoked.
// Each target function in the flow is executed by a separate GOROUTINE either parallelly or one after another
func (fl *Flow) Execute() *Flow {
	fl.runOnce.Do(func() {
		if flowFtr, err := RunAsync(fl.runFlow); err != nil {
			fmt.Errorf("error (%s) while creating future", err.Error())
		} else {
			fl.future = flowFtr
			flowFtr.SetExecutor(fl.es).Execute()
		}
	})
	return fl
}

// Call to Get() blocks until the target function(s) invocation is completed.
// Return from this method indicates successful execution of target function(s) or time-out or user aborted/cancelled this flow.
// error is returned in case of timeouts or aborted.
// This method always returns results of the last target function of the flow
func (fl *Flow) Get(timeout time.Duration) ([]interface{}, error) {
	if fl.future == nil {
		return nil, fmt.Errorf("future not created for the flow")
	}
	if timeout > 0 {
		time.AfterFunc(timeout, func() {
			for _, s := range fl.steps {
				if s.future != nil && s.future.Stage() != COMPLETED {
					fmt.Errorf("Timeout (%d) got triggered hence cancelling target (%+v)", timeout, reflect.TypeOf(s.targetFunc))
					s.future.Cancel()
				}
			}
		})
	}
	if get, e := fl.future.Get(timeout); e != nil {
		return nil, e
	} else {
		if get[1] != nil {
			return nil, get[1].(error)
		}
		return get[0].([]interface{}), nil
	}
}

// Cancel's the referred flow, sends a signal to abort the call of all the referred target function(s).
// This method returns immediately.
// Call to Cancel() does not mean the target function(s) is aborted if it is running already.
// Target function(s) will not be called if it is not triggered yet when Cancel() is called.
// Any call to *Flow.Get() after Cancel() is invoked will return an error indicating aborted
func (fl *Flow) Cancel() {
	if fl.future == nil {
		return
	}
	fl.future.Cancel()
	for _, s := range fl.steps {
		if s.future != nil {
			s.future.Cancel()
		}
	}
}
