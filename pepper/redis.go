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
	listen()
}

func listen() {
	pubsub := rdb.Subscribe(ctx, "pepper-tests")
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		test := common.TestRequest{}
		err = json.Unmarshal([]byte(msg.Payload), &test)
		if err != nil {
			panic(err)
		}

		if test.TestType == common.TestTypeInputOutput {
			innerInputOutputTest := common.InnerInputOutputTest{}
			err = json.Unmarshal([]byte(test.Tests), &innerInputOutputTest)
			if err != nil {
				panic(err)
			}
			fmt.Println(innerInputOutputTest)
		}

		// Create VM
		StartVM(test.UserProgram, test)
	}
}

func sendInnerTestResult(testId string, testIndex int, result bool) {
	innerTestOutput := common.InnerTestResult{
		ID:    testId,
		Index: testIndex,
		Ok:    result,
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
