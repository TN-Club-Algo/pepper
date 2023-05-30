package main

import (
	"AlgoTN/common"
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func Connect(address string, password string) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       0,
	})
	fmt.Println("Connected to Redis.")
	go listen()
}

func listen() {
	pubsub := rdb.Subscribe(ctx, "pepper-tests")
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		fmt.Println("Received test raw:", msg.Payload)

		test := common.TestRequest{}
		err = json.Unmarshal([]byte(msg.Payload), &test)
		if err != nil {
			panic(err)
		}

		fmt.Println("Received test request:", test)

		if test.TestType == common.TestTypeInputOutput {
			innerInputOutputTest := test.Tests
			//err = json.Unmarshal([]byte(test.Tests), &innerInputOutputTest)
			if err != nil {
				panic(err)
			}
			fmt.Println(innerInputOutputTest)
		}

		// Create VM
		go StartVM(test.ProgramLocation, test)
	}
}

func sendInnerTestResult(testId string, testIndex int, result bool) {
	innerTestOutput := common.InnerTestResult{
		ID:     testId,
		Index:  testIndex,
		Answer: "answer",
		Ok:     result,
	}

	bytes, err := json.Marshal(innerTestOutput)
	if err != nil {
		panic(err)
	}

	err = rdb.Publish(ctx, "pepper-inner-test-results", bytes).Err()
	if err != nil {
		panic(err)
	}
}

func sendTestResult(testId string, allPassed bool) {
	testResult := common.TestResult{
		ID: testId,
		Ok: allPassed,
	}

	bytes, err := json.Marshal(testResult)
	if err != nil {
		panic(err)
	}

	err = rdb.Publish(ctx, "pepper-test-results", bytes).Err()
	if err != nil {
		panic(err)
	}
}
