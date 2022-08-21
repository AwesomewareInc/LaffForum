package database

import "golang.org/x/crypto/bcrypt"

// todo: what the fuck why don't i just use string comparison huh
func TwoPasswordsMatch(pass1, pass2 string) string {
	if string(pass1) != string(pass2) {
		return "Passwords don't match!"
	}
	return ""
}

func VerifyPassword(username, password string) string {
	var hashedPass string
	ExecuteReturn("SELECT password from `users` WHERE username = ? LIMIT 1;", []interface{}{username}, &hashedPass)

	err = bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(password))

	if err != nil {
		return "Incorrect username or password."
	}

	return ""
}