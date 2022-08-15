package main

import "fmt"

// Functions for working with types.

func String(a int) (string) {
	return fmt.Sprint(a)
}
func IsInt(val any) (bool) {
	_, ok := val.(int)
	return ok
}
func IsString(val any) (bool) {
	_, ok := val.(string)
	return ok
}