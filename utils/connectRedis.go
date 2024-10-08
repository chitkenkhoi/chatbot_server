package utils

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "redis-14974.c299.asia-northeast1-1.gce.redns.redis-cloud.com:14974", // Address of your Redis server
		Password: "g3jblIiOYv3wtwnUyqPeJOgdOTjAPiDJ",                                   // If your Redis server requires a password
		DB:       0,                                                                    // Select the database (default is 0)
	})
	pong, err := client.Ping(context.TODO()).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(pong) // Output: PONG
	return client
}
