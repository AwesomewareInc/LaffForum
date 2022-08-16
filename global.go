package main

import (
	"fmt"
)

// File for global values that can be used across all files without
// extra setup.

type GlobalValues struct {
	*Session
}

func (session *GlobalValues) Username() string {
	username := session.Get("username")
	if username == nil {
		return ""
	} else {
		return *username
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
	return session.values["username"]
}

/*func (values GlobalValues) Set(key, value string) string {
	values.PhpSession[key] = value
	return values.PhpSession[key].(string)
}*/
