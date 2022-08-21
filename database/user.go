package database

import (
	"fmt"
	"time"

	"github.com/IoIxD/LaffForum/debug"
	"golang.org/x/crypto/bcrypt"
)

type UserInfo struct {
	ID         		int
	Username   		string
	PrettyName 		string
	Timestamp  		int
	bio        		interface{}
	admin      		interface{}
	deleted    		interface{}
	deletedTime 	interface{}
	Error error
}

func (user UserInfo) Bio() string {
	if user.bio == nil {
		return ""
	} else {
		return user.bio.(string)
	}
}

func (user UserInfo) Admin() bool {
	if user.admin == nil {
		return false
	} else {
		return (int(user.admin.(int64)) == 1)
	}
}

func (user UserInfo) Deleted() bool {
	if user.deleted == nil {
		return false
	} else {
		return (int(user.deleted.(int64)) == 1)
	}
}
func (user UserInfo) DeletedTime() int {
	if user.deletedTime == nil {
		return -1
	} else {
		return int(user.deletedTime.(int64))
	}
}

func GetUserInfo(id interface{}) (result UserInfo) {
	var userID int
	switch v := id.(type) {
	case string:
		j := GetUserIDByName(id.(string))
		if j.Error != nil {
			result.Error = fmt.Errorf("Couldn't get user id from username; %v", j.Error.Error())
			return
		}
		userID = int(j.Result.(int64))
	case int:
		userID = id.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
	}
	err := ExecuteReturn("SELECT id, username, prettyname, timestamp, bio, admin, deleted, deletedtime from `users` WHERE id = ?;", []interface{}{userID},
		&result.ID, &result.Username, &result.PrettyName, &result.Timestamp, &result.bio, &result.admin, &result.deleted, &result.deletedTime)
	if err != nil {
		result.Error = fmt.Errorf("Couldn't get user info; %v", err.Error())
		return
	}
	return
}

func UserExists(username string) string {
	var id int
	err := ExecuteReturn("SELECT count(id) from `users` WHERE username = ?;", []interface{}{username}, &id)
	if err != nil {
		return "Couldn't validate username; " + err.Error()
	}
	if id != 0 {
		return "Username is taken!"
	}
	return ""
}

func CreateUser(username, prettyname, pass1, pass2 string) string {
	// Check if somebody with that username exists.
	err_ := UserExists(username)
	if err_ != "" {
		return err_
	}

	// Check for invalid characters in the username
	invalidChars := usernameCheck.FindAll([]byte(username), 99)
	if len(invalidChars) >= 1 {
		return "Invalid characters in username! You can only have alphabetical letters, numbers, underscores and dashes."
	}

	// Check for invalid length of username
	if len(username) == 0 {
		return "Username cannot be blank!"
	}
	if len(username) > 21 {
		return "Username cannot be over 21 characters."
	}

	// Check if the password is blank
	if len(pass1) == 0 || len(pass2) == 0 {
		return "Passwords cannot be blank!"
	}
	// Check if the two password fields match.
	err_ = TwoPasswordsMatch(pass1, pass2)
	if err_ != "" {
		return err_
	}

	// Those are the main checks for now, now create the user.

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass1), 10)
	if err != nil {
		return "Couldn't generate password; " + err.Error()
	}
	statement, err := database.Prepare("INSERT INTO `users` (username, password, prettyname, timestamp) VALUES (?, ?, ?, ?);")
	if err != nil {
		return "Couldn't prepare statement to create user; " + err.Error()
	}
	defer statement.Close()
	_, err = statement.Exec(username, hashedPass, prettyname, fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		return "Couldn't create user; " + err.Error()
	}

	return ""
}

func GetUserIDByName(name string) (result GenericResult) {
	result.Result = -1
	err := ExecuteReturn("SELECT id from `users` WHERE username = ?;", []interface{}{name}, &result.Result)
	if err != nil {
		result.Error = debug.PublicFacingError("Error while getting user id by name;", err)
		return
	}
	return
}

func GetUsernameByID(id int) (result GenericResult) {
	err := ExecuteReturn("SELECT username from `users` WHERE id = ?;", []interface{}{id}, &result.Result)
	if err != nil {
		result.Error = debug.PublicFacingError("Error while getting username by id;", err)
		return
	}
	return
}

func (session *Session) EditProfile(prettyname, bio string) (err error) {
	err = session.Verify()
	if(err != nil) {
		return
	}

	statement, err := database.Prepare("UPDATE `users` SET prettyname = ?, bio = ? WHERE username = ?;")
	if err != nil {
		return fmt.Errorf("Couldn't prepare statement to edit user profile; " + err.Error())
	}
	defer statement.Close()
	_, err = statement.Exec(SQLSanitize(prettyname), SQLSanitize(bio), SQLSanitize(session.Username))
	if err != nil {
		return fmt.Errorf("Couldn't edit user profile; " + err.Error())
	}

	return nil
} 

func (session *Session) DeleteProfile(password string) (err error) {
	return session.SetProfileDeleteStatus(password,1,time.Now().Unix())
}
func (session *Session) UndeleteProfile(password string) (err error) {
	return session.SetProfileDeleteStatus(password,0,-1)
}

func (session *Session) SetProfileDeleteStatus(password string, status int, deletedTime int64) (err error) {
	err = session.Verify()
	if(err != nil) {
		return
	}

	if passerr := VerifyPassword(session.Username, password); passerr != "" {
		return fmt.Errorf(passerr)
	}

	statement, err := database.Prepare("UPDATE `users` SET deleted = ?, deletedtime = ? WHERE username = ?;")
	if err != nil {
		return fmt.Errorf("Couldn't prepare statement to deactivate/reactivate user; " + err.Error())
	}
	defer statement.Close()
	_, err = statement.Exec(status,deletedTime,session.Username)
	if err != nil {
		return fmt.Errorf("Couldn't deactivate/reactivate user; " + err.Error())
	}

	return nil
} 