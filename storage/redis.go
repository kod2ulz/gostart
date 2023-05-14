package storage

import (
	"strconv"

	"github.com/go-redis/redis/v8"
)

func Redis(conf *Conf) *redis.Client {
	database, _ := strconv.Atoi(conf.Database)
	return redis.NewClient(&redis.Options{
		Addr:     conf.Host + ":" + conf.Port,
		Password: conf.Password, 
		DB:       database,      
	})
}