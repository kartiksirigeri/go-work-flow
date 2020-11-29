package workflow

import (
	"fmt"
	"testing"
	"time"
)

func TestFlow_AndCall(t *testing.T) {

	counter := 2
	flow := NewFlow(func() {
		counter = counter - 1
	}).AndCall(func() {
		time.Sleep(50 * time.Millisecond)
		counter = counter - 1
	})
	flow.Execute()
	flow.Get(0)

	if counter != 0 {
		t.Errorf("expected counter=0 but counter=%d", counter)
	}
}

func TestFlow_ThenApply(t *testing.T) {

	counter := 1

	addFlow := NewFlow(func() int {
		return 2
	}).ThenApply(func(cnt int) {
		counter = cnt
	})
	addFlow.Execute()
	if get, err := addFlow.Get(0); err != nil {
		t.Errorf("did not expect an error (%s)", err.Error())
	} else if len(get) == 0 && counter == 2 {

	} else {
		t.Errorf("expected a return value of 2 but got %d", counter)
	}
}

func TestFlow_ThenCombine(t *testing.T) {

	addFlow := NewFlow(func() int {
		return 1
	}).AndCall(func() int {
		return 1
	}).ThenCombine(func(op1, op2 int) int {
		return op1 + op2
	})
	addFlow.Execute()
	if get, err := addFlow.Get(0); err != nil {
		t.Errorf("did not expect an error (%s)", err.Error())
	} else if len(get) == 1 && get[0] == 2 {

	} else {
		t.Errorf("expected a return value of 2 but got %d", get[0])
	}

}

func TestFlow_Cancel(t *testing.T) {
	flow := NewFlow(func() bool {
		time.Sleep(100 * time.Millisecond)
		return true
	})

	flow.Execute()
	time.AfterFunc(50*time.Millisecond, func() {
		fmt.Println("Triggered user abort")
		flow.Cancel()
	})

	if _, err := flow.Get(0); err == nil {
		t.Errorf("expected an error!!!")
	}
}

func TestFlow_Get_TimeOut(t *testing.T) {
	flow := NewFlow(func() bool {
		time.Sleep(100 * time.Millisecond)
		return true
	})

	flow.Execute()
	if _, err := flow.Get(5 * time.Millisecond); err == nil {
		t.Errorf("expected an error!!!")
	}

}

func TestFlow_Execute_RunOnce(t *testing.T) {
	counter := 1
	flow := NewFlow(func() {
		counter = counter - 1
	})

	go func() {
		flow.Execute()
	}()
	go func() {
		flow.Execute()
	}()

	flow.Get(0)
	if counter < 0 {
		t.Errorf("target func ran twice!!, expected counter=0 but it is set to %d", counter)
	}
}

func ExampleNewFlow() {

	getBillAmount := func(timeDelay time.Duration) int {
		time.Sleep(timeDelay * time.Millisecond) //simulate blocking call
		return 100
	}

	getDollarValue := func(timeDelay time.Duration) int {
		time.Sleep(timeDelay * time.Millisecond) //simulate blocking call
		return 60
	}

	billInDollars := func(billAmount, conversionRate int) int {
		return billAmount * conversionRate
	}

	billFlow := NewFlow(getBillAmount, time.Duration(30)).
		AndCall(getDollarValue, time.Duration(30)).
		ThenCombine(billInDollars).
		ThenApply(func(finalAmount int) string {
			return fmt.Sprintf("Bill Amount = %d$", finalAmount)
		})

	newExecutorService := NewExecutorService(10, 5)

	billFlow.SetExecutor(newExecutorService).Execute()
	if finalAmount, e := billFlow.Get(0); e != nil {
		fmt.Errorf("error (%s) while waiting to get the final bill amount", e.Error())
	} else {
		fmt.Println(finalAmount[0])
	}

	//// Output:
	//// Bill Amount = 6000$

}
