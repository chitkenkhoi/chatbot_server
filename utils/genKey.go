package utils

import (
	"context"
	"crypto/sha256"
	"os"
	"time"
	"github.com/redis/go-redis/v9"
	"encoding/hex"
)

func GenerateToken(email string, redisClient *redis.Client) string {
	hash := sha256.New()
	hash.Write([]byte(email + os.Getenv("KEY_FOR_REGISTER")))
	hashString := hex.EncodeToString(hash.Sum(nil))
	redisClient.Set(context.TODO(), "token_"+email, hashString, 15*time.Minute)
	return hashString
}
func VerifyToken(email, token string, redisClient *redis.Client) bool {
	value, err := redisClient.Get(context.TODO(), "token_"+email).Result()
	if err != nil {
		return false
	}
	if value != token {
		return false
	}
	redisClient.Del(context.TODO(), "token_"+email)
	return true
}
