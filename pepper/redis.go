package main

import (
	"AlgoTN/common"
	"context"
	"encoding/json"
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
}
func listen() {
	pubsub := rdb.Subscribe(ctx, "pepper-tests")
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			panic(err)
		}

		test := common.Test{}
		err = json.Unmarshal([]byte(msg.Payload), &test)
		if err != nil {
			panic(err)
		}

		// Create VM
		StartVM(test.UserProgram)
	}
}
