package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var database *sql.DB
var err error
var usernameCheck *regexp.Regexp

func init() {
	database, err = sql.Open("sqlite3","database.db")
	if(err != nil) {
		fmt.Println(err)
		return
	}
	usernameCheck = regexp.MustCompile(`[^A-z0-9_-]`)
}

func UserExists(username string) (string) {
	statement, err := database.Prepare("SELECT id from users WHERE username = ?;")
	if(err != nil) {
		return err.Error()
	}
	defer statement.Close()

	rows, err := statement.Query(username)
	defer rows.Close()
	if(err != nil) {
		return err.Error()
	}
	results := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return err.Error()
		}
		results = append(results, id)
	}
	if(len(results) >= 1) {
		return "Username is taken!"
	}
	return ""
}

func TwoPasswordsMatch(pass1, pass2 string) (string) {
	if(string(pass1) != string(pass2)) {
		return "Passwords don't match!"
	}
	return ""
}

func CreateUser(username, prettyname, pass1, pass2 string) (string) {
	
	// Check if the username is allowed

	// Check if it exists.
	err_ := UserExists(username)
	if(err_ != "") {return err_}

	// Check for invalid characters in the username
	invalidChars := usernameCheck.FindAll([]byte(username),99)
	if(len(invalidChars) >= 1) {
		return "Invalid characters in username! You can only have alphabetical letters, numbers, underscores and dashes."
	} 

	// Check for invalid length of username 
	if(len(username) == 0) {
		return "Username cannot be blank!"
	}
	if(len(username) > 21) {
		return "Username cannot be over 21 characters."
	}

	// Check if the password is allowed

	// Check if they're blank
	if(len(pass1) == 0 || len(pass2) == 0) {
		return "Passwords cannot be blank!"
	}
	// Check if the two fields match. 
	err_ = TwoPasswordsMatch(pass1, pass2)
	if(err_ != "") {return err_}

	// Those are the main checks for now, now create the user.


	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass1),10)
	if(err != nil) {
		return err.Error()
	}
	statement, err := database.Prepare("INSERT INTO users (username, password, prettyname, timestamp) VALUES (?, ?, ?, ?);")
	if(err != nil) {
		return err.Error()
	}
	defer statement.Close()
	_, err = statement.Exec(username,hashedPass,prettyname,fmt.Sprintf("%v",time.Now().Unix()))
	if(err != nil) {
		return err.Error()
	}
	
	return ""
}

func VerifyPassword(username, password string) (string) {
	row := database.QueryRow("SELECT password from users WHERE username = ?;",username)
	var hashedPass string
	err = row.Scan(&hashedPass) 

	err = bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(password))

	if(err != nil) {
		return "Incorrect username or password."
	}

	return ""
}


