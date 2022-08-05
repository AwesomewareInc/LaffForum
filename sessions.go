package main

import (
	"os"

	"github.com/redis-go/redis"
	"github.com/tiket-oss/phpsessgo/phpencode"
)

func init() {
	go func() {
		err := redis.Run("localhost:6379")
		if(err != nil) {
			os.Exit(0);
		}
	}()
}

func SetSessionValue(theMap phpencode.PhpSession, key, value string) (string) {
	theMap[key] = value;
	return theMap[key].(string);
};