package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var DatabaseConfig struct {
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
}

var database *sql.DB
var err error
var usernameCheck *regexp.Regexp

var SQLEscape = strings.NewReplacer(
	";", "\\;",
	"\\", "\\\\",
	"'", "\\'",
	"\"", "\\\"",
	"\x00", "\\\x00",
	"\x1a", "\\\x1a",
)

func init() {
	// Load the database
	database, err = sql.Open("sqlite3", "database.db"+
		"?_busy_timeout=10000"+
		"&_journal=WAL"+
		"&_sync=NORMAL"+
		"&cache=shared")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// 9 times out of ten that error doesnt do shit for us lol
	_, err = database.Exec(`;`)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Try and create the database.
	file, err := os.Open("./tableStructure.sql")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sql := make([]byte,2048)
	_, err = file.Read(sql)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	statements := strings.Split(string(sql),";")
	for _, v := range statements {
		err = ExecuteDirect(v)
		if err != nil {
			// If-oh. fuck.
			if(!strings.Contains(err.Error(),"duplicate") && !strings.Contains(err.Error(),"more than one primary key")) {
				fmt.Println(err)
				os.Exit(1)
			} 
		}
	}


	usernameCheck = regexp.MustCompile(`[^A-z0-9_-]`)

	// Start listening for commands
	go CommandListenerThread()
}

func CommandListenerThread() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">")
		text, _ := reader.ReadString('\n')
		err := DoCommand(text)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// =============
// GENERAL SHIT
// =============

func ExecuteDirect(query string, args ...any) error {
	_, err = database.Exec(query, args[:]...)
	return err
}

func ExecuteReturn(query string, args []any, dest ...any) error {
	rows, err := database.Query(query, args[:]...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return err
		}
	}
	return nil
}

type GenericResult struct {
	Result any
	Error  error
}

func SQLSanitize(val string) string {
	return SQLEscape.Replace(val)
}

// ==============
// USER SHIT
// ==============

type UserInfo struct {
	ID         int
	Username   string
	PrettyName string
	Timestamp  int
	bio        interface{}
	admin      interface{}

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
	err := ExecuteReturn("SELECT id, username, prettyname, timestamp, bio, admin from `users` WHERE id = ?;", []interface{}{userID},
		&result.ID, &result.Username, &result.PrettyName, &result.Timestamp, &result.bio, &result.admin)
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
		result.Error = PublicFacingError("Error while getting user id by name;", err)
		return
	}
	return
}

func GetUsernameByID(id int) (result GenericResult) {
	err := ExecuteReturn("SELECT username from `users` WHERE id = ?;", []interface{}{id}, &result.Result)
	if err != nil {
		result.Error = PublicFacingError("Error while getting username by id;", err)
		return
	}
	return
}

// ==============
// PASSWORD SHIT
// ==============

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

// ==============
// SECTION SHIT
// ==============

type Section struct {
	ID        int
	Name      string
	AdminOnly int

	Error error
}

type GetSectionsResult struct {
	Results []Section
	Error   error
}

func GetSections() (result GetSectionsResult) {
	result.Results = make([]Section, 0)
	statement, err := database.Prepare("SELECT id, name, adminonly from `sections`;")
	if err != nil {
		result.Error = err
		return
	}
	defer statement.Close()

	rows, err := statement.Query()
	for rows.Next() {
		var id int
		var name string
		var adminonly int
		if err := rows.Scan(&id, &name, &adminonly); err != nil {
			result.Error = err
			return
		}
		result.Results = append(result.Results, Section{id, name, adminonly, nil})
	}
	return result
}

func GetSectionInfo(id interface{}) (result Section) {
	var sectionID int
	switch v := id.(type) {
	case string:
		j := GetSectionIDByName(id.(string))
		if j.Error != nil {
			result.Error = fmt.Errorf("Couldn't get info for the %v secion; %v", id, j.Error.Error())
			return
		}
		sectionID = int(j.Result.(int64))
	case int:
		sectionID = id.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
	}
	err := ExecuteReturn("SELECT id, name, adminonly from `sections` WHERE id = ?;", []interface{}{sectionID},
		&result.ID, &result.Name, &result.AdminOnly)
	if err != nil {
		result.Error = fmt.Errorf("Couldn't get info for the %v secion; %v", id, err.Error())
		return
	}
	return
}

func GetSectionIDByName(name string) (result GenericResult) {
	err := ExecuteReturn("SELECT id from `sections` WHERE name = ?;", []interface{}{name}, &result.Result)
	if err != nil {
		result.Error = PublicFacingError("Error while getting section id by name;", err)
		return
	}
	return
}

// ==============
// POST SHIT
// ==============

type Post struct {
	ID        int
	Topic     int
	Subject   string
	Contents  string
	Author    int
	ReplyTo   int
	Timestamp int
	deleted   interface{}

	Error error
}

func (p Post) Deleted() bool {
	if p.deleted == nil {
		return false
	} else {
		return (p.deleted.(int64) == 1)
	}
}

func GetPostInfo(id interface{}) (result Post) {
	var postID int
	var err error
	switch v := id.(type) {
	case string:
		postID_ := id.(string)
		postID, err = strconv.Atoi(postID_)
		if err != nil {
			result.Error = PublicFacingError("Error while getting post info;", err)
			return
		}
	case int:
		postID = id.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
	}

	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp, deleted from `posts` WHERE id = ? LIMIT 1;")
	if err != nil {
		result.Error = PublicFacingError("Error while getting post info;", err)
		return
	}
	defer statement.Close()

	rows, err := statement.Query(postID)
	for rows.Next() {
		if err := rows.Scan(&result.ID,
			&result.Topic,
			&result.Subject,
			&result.Contents,
			&result.Author,
			&result.ReplyTo,
			&result.Timestamp,
			&result.deleted); err != nil {
			result.Error = PublicFacingError("Error while getting post info;", err)
		}
	}

	return
}

type GetPostsByCriteriaResult struct {
	Posts []Post
	Error error
}

func GetPostsByCriteria(criteria string, value any) (result GetPostsByCriteriaResult) {
	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp, deleted FROM `posts` " + criteria)
	if err != nil {
		result.Error = fmt.Errorf("Couldn't get posts following the relevant criteria; " + err.Error())
		return
	}
	defer statement.Close()

	rows, err := statement.Query(value)
	for rows.Next() {
		var id interface{}
		var topic int
		var subject string
		var contents string
		var author int
		var replyto int
		var timestamp int
		var deleted interface{}
		if err := rows.Scan(&id, &topic, &subject, &contents, &author, &replyto, &timestamp, &deleted); err != nil {
			result.Error = PublicFacingError("Error getting posts by section name; ", err)
			return
		}
		result.Posts = append(result.Posts, Post{int(id.(int64)), topic, subject, contents, author, replyto, timestamp, deleted, nil})
	}
	return
}

func GetPostsBySectionName(name string) (result GetPostsByCriteriaResult) {
	id_ := GetSectionIDByName(name)
	if id_.Error != nil {
		result.Error = PublicFacingError("Error getting posts by section name; ", id_.Error)
		return
	}
	if(id_.Result == nil) {
		return
	}
	id := id_.Result.(int64)

	result = GetPostsByCriteria("WHERE topic = ? AND replyto = 0 ORDER BY id DESC;", id)

	return
}

func GetPostsFromUser(name string) (result GetPostsByCriteriaResult) {
	id_ := GetUserIDByName(name)
	if id_.Error != nil {
		result.Error = PublicFacingError("Error getting posts from user; ", id_.Error)
		return
	}
	id := id_.Result.(int64)

	result = GetPostsByCriteria("WHERE author = ? ORDER BY id DESC;", id)

	return
}

func GetPostsInReplyTo(id int) (result GetPostsByCriteriaResult) {
	posts := GetPostsByCriteria("WHERE replyto = ?;", id)
	if posts.Error != nil {
		return
	}
	for _, v := range posts.Posts {
		posts_ := GetPostsInReplyTo(v.ID)
		if posts_.Error != nil {
			return posts_
		}
		for _, v_ := range posts_.Posts {
			posts.Posts = append(posts.Posts, v_)
		}
	}
	sort.Slice(posts.Posts[:], func(i, j int) bool {
		return posts.Posts[i].Timestamp < posts.Posts[j].Timestamp
	})
	return posts
}

func GetLastTenPostsMadeAtAll() (result GetPostsByCriteriaResult) {
	return GetPostsByCriteria("ORDER BY id DESC LIMIT 10;",nil)
}

func (session *Session) DeletePost(r *http.Request, id interface{}, deletedBy string) (err error) {
	return session.SetDeleteStatus(r, id, deletedBy, 1)
}
func (session *Session) RestorePost(r *http.Request, id interface{}, deletedBy string) (err error) {
	return session.SetDeleteStatus(r, id, deletedBy, 0)
}
func (session *Session) SetDeleteStatus(r *http.Request, id interface{}, deletedBy string, deleteStatus int) (err error) {
	// Check the "session" that wants to modify this post.
	err = session.Verify(r)
	if(err != nil) {
		return
	}

	var postID int
	switch v := id.(type) {
	case string:
		postID_ := id.(string)
		postID, err = strconv.Atoi(postID_)
		if err != nil {
			return
		}
	case int:
		postID = id.(int)
	default:
		return fmt.Errorf("Invalid type '%v' given.", v)
	}

	err = ExecuteDirect("UPDATE `posts` SET deleted = ? WHERE id = ?", deleteStatus, postID)
	if err != nil {
		return
	}
	return nil
}

type SubmitPostResult struct {
	ID     int64
	Result Post
	Error  error
}

func (session *Session) SubmitPost(r *http.Request, topic interface{}, subject string, content string, replyto interface{}) (result *SubmitPostResult) {
	result = new(SubmitPostResult)

	// Check the "session" that submitted this post.
	err := session.Verify(r)
	if(err != nil) {
		result.Error = fmt.Errorf("Verification error; %v",err)
		return
	}

	
	
	// topic/reply IDs from string, if we have to.
	var topicID int
	switch v := topic.(type) {
	case string:
		j := GetSectionIDByName(topic.(string))
		if j.Error != nil {
			result.Error = fmt.Errorf("Couldn't get info for the %v section; %v", topic, j.Error.Error())
			return
		}
		topicID = int(j.Result.(int64))
	case int:
		topicID = topic.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
	}
	var replyID int
	switch v := replyto.(type) {
	case string:
		replyID, err = strconv.Atoi(replyto.(string))
		if err != nil {
			result.Error = err
			return
		}
	case int:
		replyID = replyto.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
	}

	//

	// Check for invalid length of things
	if len(subject) == 0 {
		result.Error = fmt.Errorf("Subject cannot be blank")
		return
	}
	if len(content) == 0 {
		result.Error = fmt.Errorf("Contents of the post cannot be blank")
		return
	}

	// Prepare to insert into posts.
	statement, err := database.Prepare("INSERT INTO `posts` (author, topic, subject, contents, timestamp, replyto) VALUES (?, ?, ?, ?, ?, ?);")
	if err != nil {
		result.Error = err
		return
	}
	defer statement.Close()

	// Get the necessary values.
	userid_ := GetUserIDByName(session.Username)
	if userid_.Error != nil {
		result.Error = PublicFacingError("", userid_.Error)
		return
	}
	userid := userid_.Result.(int64)

	timestampInt := time.Now().Unix()
	timestamp := fmt.Sprintf("%v", timestampInt)

	// Submit the post with those values and what we got in the function arguments, and return the new post id.
	execResult, err := statement.Exec(userid, topicID, subject, content, timestamp, replyID)
	if err != nil {
		result.Error = PublicFacingError("", err)
		return
	}
	result.Result = GetPostInfo(fmt.Sprint(result.ID))
	result.ID, err = execResult.LastInsertId()
	if err != nil {
		result.Error = PublicFacingError("", err)
		return
	}

	return
}

// ==============
// ADMIN SHIT
// ==============

func DoCommand(text string) error {
	text = strings.Replace(text, "\n", "", 99)
	args := strings.Split(text, " ")
	command := args[0]
	var err error
	switch command {
	case "createsection":
		if len(args) < 3 {
			return fmt.Errorf("createsection <string sectionname> <int adminonly>")
		}
		err = CreateSection(args[1], args[2])
	case "mkadmin":
		if len(args) < 2 {
			return fmt.Errorf("mkadmin <string username>")
		}
		err = MakeAdmin(args[1])
	case "exit":
		os.Exit(0)
	default:
		fmt.Println("Invalid command")
	}
	return err
}

func CreateSection(args ...any) error {
	return ExecuteDirect("INSERT INTO `sections` (name, adminonly) VALUES (?, ?);", args...)
}

func MakeAdmin(args ...any) error {
	return ExecuteDirect("UPDATE `users` SET admin = 1 WHERE username = ?;", args...)
}
