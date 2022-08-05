package main

import (
	"github.com/tiket-oss/phpsessgo/phpencode"
)

// here's a function for setting session values since i guess golang doesn't have that
func SetSessionValue(theMap phpencode.PhpSession, key, value string) (string) {
	theMap[key] = value;
	return theMap[key].(string);
};
//
