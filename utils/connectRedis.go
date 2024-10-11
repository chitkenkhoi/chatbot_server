package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"), // Address of your Redis server
		Password: os.Getenv("REDIS_PASS"),                                   // If your Redis server requires a password
		DB:       0,                                                                    // Select the database (default is 0)
	})
	pong, err := client.Ping(context.TODO()).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong) // Output: PONG
	return client
}
