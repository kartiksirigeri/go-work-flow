package workflow

import (
	"fmt"
	"testing"
	"time"
)

func TestExecutorService_MaxConcurrentTasks(t *testing.T) {
	toTest := NewExecutorService(2, 1)
	toTest.Submit(func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Task 1 Running")
	})
	toTest.Submit(func() {
		fmt.Println("Task 2 Running")
	})

	<-time.After(75 * time.Millisecond)
	waiting := len(toTest.tasksQueue)
	if waiting != 1 {
		t.Errorf("expected only 1 task waiting but got %d tasks waiting", waiting)
	}

	<-time.After(50 * time.Millisecond)
	waiting = len(toTest.tasksQueue)
	if waiting != 0 {
		t.Errorf("expected 0 tasks waiting but got %d tasks waiting", waiting)
	}
}

func ExampleExecutorService_Submit_defaultService() {
	add := func(operands ...int) int {
		sum := 0

		for _, operand := range operands {
			sum = sum + operand
		}

		return sum
	}

	default_es.Submit(func() {
		summedUp := add(1, 2, 3, 4)
		fmt.Printf("1+2+3+4=%d", summedUp)
	})
	time.Sleep(1 * time.Nanosecond) //Need at least 1 NanoSecond sleep otherwise test would fail
	//// Output:
	//// 1+2+3+4=10
}

func ExampleNewExecutorService() {
	add := func(operands ...int) int {
		sum := 0

		for _, operand := range operands {
			sum = sum + operand
		}

		return sum
	}

	maxQueueSize := 1
	maxGoRoutines := 2

	newExecutorService := NewExecutorService(maxQueueSize, maxGoRoutines)
	newExecutorService.Submit(func() {
		summedUp := add(1, 2, 3, 4)
		fmt.Printf("1+2+3+4=%d", summedUp)
	})
	time.Sleep(10 * time.Nanosecond) //Need at least 1 NanoSecond sleep otherwise test would fail
	//// Output:
	//// 1+2+3+4=10

}
