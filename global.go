package main

import (
	"github.com/tiket-oss/phpsessgo/phpencode"
)

// File for global values that can be used across all files without
// extra setup.

type GlobalValues struct {
	phpencode.PhpSession
}

func (values GlobalValues) Username() string {
	if values.PhpSession["username"] == nil {
		return ""
	} else {
		return values.PhpSession["username"].(string)
	}
}

func (values GlobalValues) Me() UserInfo {
	if values.Username() == "" {
		return UserInfo{}
	} else {
		return GetUserInfo(values.Username())
	}
}

func (values GlobalValues) SetUsername(value string) string {
	values.PhpSession["username"] = value
	return values.PhpSession["username"].(string)
}

/*func (values GlobalValues) Set(key, value string) string {
	values.PhpSession[key] = value
	return values.PhpSession[key].(string)
}*/
