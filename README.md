# go-work-flow
GO-WORK-FLOW is a small, simple utility to run methods/functions in parallel, create a pipeline/chain of methods, combine the results of mutilple parallel method/function calls and do some processing on the returns from a method that is run asyncrhonously.

The utility helps to do all of this in a intuitive way.

Example usage:

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

In the example above each of the function getBillAmount, getDollarValue and billInDollars all run asynchronously. 
