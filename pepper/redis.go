package main

import (
	"AlgoTN/common"
	"context"
	"encoding/json"
	"github.com/pbnjay/memory"
	"github.com/redis/go-redis/v9"
	"log"
	"strconv"
)

var (
	ctx = context.Background()
	rdb *redis.Client

	TestQueue = make(chan common.TestRequest)
)

func Connect(address string, password string) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       0,
	})
	log.Println("Connected to Redis.")
	go listen()
}

func listen() {
	pubsub := rdb.Subscribe(ctx, "pepper-tests")
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err == nil {
			log.Println("Received test raw:", msg.Payload)

			test := common.TestRequest{}
			err = json.Unmarshal([]byte(msg.Payload), &test)

			if err == nil {
				if common.SumMapValues(ActiveVMs) > MaxRam || memory.FreeMemory() < 4096 {
					log.Println("Not enough RAM to start VM, waiting...")
					TestQueue <- test
					continue
				}

				// Create VM
				go StartVM(test.ProgramURL, test)
			}
		}
	}
}

func sendInnerTestResult(testId string, testIndex int, problemSlug string, result string, timeElapsed int,
	memoryUsed int, sendFinalResult bool, finalPassed bool) {
	innerTestOutput := common.InnerTestResult{
		ID:          testId,
		Index:       testIndex,
		ProblemSlug: problemSlug,
		Result:      result,
		TimeElapsed: timeElapsed,
		MemoryUsed:  memoryUsed,
	}

	bytes, err := json.Marshal(innerTestOutput)
	if err != nil {
		return
	}

	if sendFinalResult {
		go sendTestResult(testId, problemSlug, result, finalPassed)
	} else {
		err = rdb.Publish(ctx, "pepper-inner-test-results", string(bytes)).Err()
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func sendTestResult(testId string, problemSlug string, info string, allPassed bool) {
	testResult := common.TestResult{
		ID:          testId,
		ProblemSlug: problemSlug,
		Info:        info,
		Result:      strconv.FormatBool(allPassed),
	}

	bytes, err := json.Marshal(testResult)
	if err != nil {
		return
	}

	err = rdb.Publish(ctx, "pepper-test-results", string(bytes)).Err()
	if err != nil {
		log.Println(err)
		return
	}
}
