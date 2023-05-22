package main

import "github.com/redis/go-redis/v9"

var (
	rdb *redis.Client
)

func Connect(address string, password string) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       0,
	})
}
