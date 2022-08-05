package main

import (
	"fmt"

	"github.com/redis-go/redis"
	"github.com/tiket-oss/phpsessgo/phpencode"
)

// yeah there's a redis server needed for the php session package. 
// the implementation is old and crusty but it works so ¯\_(ツ)_/¯
func init() {
	go func() {
		err := redis.Run("localhost:6379")
		if(err != nil) {
			fmt.Println(err)
			fmt.Println("If you're already running a Redis server, you can feel free to ignore it")
		}
	}()
}

// here's a function for setting session values since i guess golang doesn't have that
func SetSessionValue(theMap phpencode.PhpSession, key, value string) (string) {
	theMap[key] = value;
	return theMap[key].(string);
};