package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedis(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-12878.c1.ap-southeast-1-1.ec2.redns.redis-cloud.com:12878", // Redis server address
		Password: "qKl6znBvULaveJhkjIjMr7RCwluJjjbH",                                // No password set
		DB:       0,                                                                 // Use default DB
	})
	ctx := context.Background()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		fmt.Println("Error connecting to Redis:", err)
		return
	}
	fmt.Println("Connected to Redis:", pong)
	err = rdb.Set(ctx, "mykey", "Hello Redis", 0).Err()
	if err != nil {
		fmt.Println("Error setting key:", err)
		return
	}
	fmt.Println("Key set successfully")

	// GET operation
	val, err := rdb.Get(ctx, "mykey").Result()
	if err != nil {
		fmt.Println("Error getting key:", err)
		return
	}
	fmt.Println("Value of 'mykey':", val)
}
