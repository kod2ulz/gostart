package storage

import (
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/kod2ulz/gostart/config"
)

func Redis(prefix ...string) *redis.Client {
	// Helper to construct keys with the given prefix
	buildKey := func(key string) string {
		return strings.ToUpper(strings.Join(append(prefix, key), "_"))
	}

	host := config.Get(buildKey("HOST"), "localhost").String()
	port := config.Get(buildKey("PORT"), "6379").String()
	password := config.Get(buildKey("PASSWORD"), "").String()
	db := config.Get(buildKey("DATABASE"), "0").Int()

	return redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
}
