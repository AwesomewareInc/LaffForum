package main


// File for global values that can be used across all files without
// extra setup.

type GlobalValues struct {
	*Session
}

func (session *GlobalValues) Username() string {
	return session.get("username")
}

func (session *GlobalValues) Me() UserInfo {
	if session.Username() == "" {
		return UserInfo{}
	} else {
		return GetUserInfo(session.Username())
	}
}

func (session *GlobalValues) SetUsername(value string) string {
	session.set("username", value)
	return session.get("username")
}

/*func (values GlobalValues) Set(key, value string) string {
	values.PhpSession[key] = value
	return values.PhpSession[key].(string)
}*/
