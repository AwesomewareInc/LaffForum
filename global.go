package main

import (
	"fmt"
	"sync"

	"github.com/tiket-oss/phpsessgo/phpencode"
)

// File for global values that can be used across all files without
// extra setup.

type GlobalValues struct {
	values phpencode.PhpSession
	mutex sync.Mutex
}

func (session *GlobalValues) Username() string {
	session.mutex.Lock()
	fmt.Println(session.values["username"])
	if session.values["username"] == nil {
		session.mutex.Unlock()
		return ""
	} else {
		session.mutex.Unlock()
		return session.values["username"].(string)
	}
}

func (session *GlobalValues) Me() UserInfo {
	if session.Username() == "" {
		return UserInfo{}
	} else {
		return GetUserInfo(session.Username())
	}
}

func (session *GlobalValues) SetUsername(value string) string {
	fmt.Println(value)
	session.mutex.Lock()
	session.values["username"] = value
	session.mutex.Unlock()
	return session.values["username"].(string)
}

/*func (values GlobalValues) Set(key, value string) string {
	values.PhpSession[key] = value
	return values.PhpSession[key].(string)
}*/
