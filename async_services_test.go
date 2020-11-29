package workflow

import (
	"fmt"
	"testing"
	"time"
)

func TestFuture_Execute_VariadicArgs(t *testing.T) {

	if future, err := RunAsync(func(operands ...int) int {
		var sum = 0
		for _, operand := range operands {
			sum = sum + operand
		}
		return sum
	}, 1, 1); err != nil {
		t.Errorf("error (%s) creating future", err.Error())
	} else {
		future.Execute()
		if get, err := future.Get(0); err != nil {
			t.Errorf(err.Error())
		} else if get[0] != 2 {
			t.Errorf("expected 2 but got %d", get[0])
		}
	}
}

func TestFuture_Execute_RunOnce(t *testing.T) {
	counter := 1

	if future, err := RunAsync(func() {
		counter = counter - 1
	}); err != nil {
		t.Errorf("error (%s) creating future", err.Error())
	} else {
		go func() {
			future.Execute()
		}()
		go func() {
			future.Execute()
		}()
		<-time.After(1 * time.Second)
		if counter < 0 {
			t.Errorf("target func ran twice!!, expected counter=0 but it is set to %d", counter)
		}
	}
}

func TestFuture_Cancel(t *testing.T) {

	if future, err := RunAsync(func() bool {
		time.Sleep(100 * time.Millisecond)
		return true
	}); err != nil {
		t.Errorf("error (%s) creating future", err.Error())
	} else {
		future.Execute()

		go func() {
			future.Cancel()
		}()

		if _, err := future.Get(0); err == nil {
			t.Errorf("expected an error!!!")
		}
	}

}

func TestFuture_Get_TimeOut(t *testing.T) {
	if future, err := RunAsync(func() bool {
		time.Sleep(100 * time.Millisecond)
		return true
	}); err != nil {
		t.Errorf("error (%s) creating future", err.Error())
	} else {

		future.Execute()
		if _, err := future.Get(5 * time.Millisecond); err == nil {
			t.Errorf("expected an error!!!")
		}

	}
}

func TestFuture_Execute_NilMethodParam(t *testing.T) {
	if future, err := RunAsync(func(err error) bool {
		return true
	}, nil); err != nil {
		t.Errorf("error (%s) creating future", err.Error())
	} else {

		future.Execute()
		if _, err := future.Get(0); err != nil {
			t.Errorf("did not expect an error!!!")
		}
	}
}

func ExampleRunAsync() {
	add := func(operands ...int) int {
		time.Sleep(50 * time.Millisecond) //simulate a blocking call
		sum := 0
		for _, operand := range operands {
			sum = sum + operand
		}

		return sum
	}

	if future, err := RunAsync(add, 1, 2, 3, 4); err != nil {
		fmt.Errorf("error (%s) creating a future for add func", err.Error())
	} else {
		future.Execute()                                                //submit the future
		if sum, err := future.Get(100 * time.Millisecond); err != nil { //timed waiting
			fmt.Errorf("error (%s) while waiting for execution completion", err.Error())
		} else {
			fmt.Println(sum[0])
		}
	}

	//// Output:
	//// 10
}
