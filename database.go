package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
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
	fmt.Printf(strings.Replace(query, "?", "%v", 99)+"\n", args[:]...)
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

func PublicFacingError(msg string, err error) error {

	// stack trace
	stacktrace := string(debug.Stack())

	// code into
	pc, filename_, line, _ := runtime.Caller(1)

	// manipulate the stacktrace.
	stacktraceParts := strings.Split(stacktrace, "\n")[3:] // the first three lines are guaranteed to be part of this call.
	var relevant bool                                      // whether we've begun encountering lines that are part of this project
	var maxStackDetail int                                 // the point at which we stop encountering those lines.
	// for each part of the stacktrace...
	for i, v := range stacktraceParts {
		// does it many slashes in it?
		if strings.Count(v, string(os.PathSeparator)) >= 2 {
			// how many tabs in it?
			tabcount := strings.Count(v, "	")
			// split it into parts and filter the line to only the last part
			stacktracePartParts := strings.Split(v, string(os.PathSeparator))
			// make sure it retains the amount of tabs
			var newString string
			for i := 0; i < tabcount; i++ {
				newString += "	"
			}
			newString += stacktracePartParts[len(stacktracePartParts)-1]
			stacktraceParts[i] = newString
		}
		if strings.Contains(v, "LaffForum") {
			if relevant == false {
				relevant = true
			} else {
				maxStackDetail = i + 3
				break
			}
		}
	}

	// and reduce the stacktrace to fit in the scope we want.
	stacktrace = strings.Join(stacktraceParts[0:maxStackDetail], "\n")
	stacktrace += "\n(...continues entering system files...)"
	filenameParts := strings.Split(filename_, "/")
	filename := filenameParts[len(filenameParts)-1]

	funcname_ := runtime.FuncForPC(pc).Name()
	funcnames := strings.Split(funcname_, ".")
	funcname := funcnames[len(funcnames)-1]

	return fmt.Errorf("%v at %v:%v in %v(), %v. \n\n%v", msg, filename, line, funcname, err.Error(), stacktrace)
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

	// Check if the username is allowed

	// Check if somebody with that name exists.
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

	// Check if the password is allowed

	// Check if they're blank
	if len(pass1) == 0 || len(pass2) == 0 {
		return "Passwords cannot be blank!"
	}
	// Check if the two fields match.
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
	err := ExecuteReturn("SELECT count(id) from `users` WHERE username = ?;", []interface{}{name}, &result.Result)
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

	Error error
}

func GetPostInfo(id_ string) (result Post) {
	id, err := strconv.Atoi(id_)
	if err != nil {
		result.Error = PublicFacingError("Error while getting post info;", err)
		return
	}

	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp from `posts` WHERE id = ? LIMIT 1;")
	if err != nil {
		result.Error = PublicFacingError("Error while getting post info;", err)
		return
	}
	defer statement.Close()

	rows, err := statement.Query(id)
	for rows.Next() {
		if err := rows.Scan(&result.ID,
			&result.Topic,
			&result.Subject,
			&result.Contents,
			&result.Author,
			&result.ReplyTo,
			&result.Timestamp); err != nil {
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
	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp FROM `posts` " + criteria)
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
		if err := rows.Scan(&id, &topic, &subject, &contents, &author, &replyto, &timestamp); err != nil {
			result.Error = PublicFacingError("Error getting posts by section name; ", err)
			return
		}
		result.Posts = append(result.Posts, Post{int(id.(int64)), topic, subject, contents, author, replyto, timestamp, nil})
	}
	return
}

func GetPostsBySectionName(name string) (result GetPostsByCriteriaResult) {
	id_ := GetSectionIDByName(name)
	if id_.Error != nil {
		result.Error = PublicFacingError("Error getting posts by section name; ", id_.Error)
		return
	}
	id := id_.Result.(int64)

	result = GetPostsByCriteria("WHERE topic = ? AND replyto = 0;", id)

	return
}

func GetPostsFromUser(name string) (result GetPostsByCriteriaResult) {
	id_ := GetUserIDByName(name)
	if id_.Error != nil {
		result.Error = PublicFacingError("Error getting posts from user; ", id_.Error)
		return
	}
	id := id_.Result.(int64)

	result = GetPostsByCriteria("WHERE id = ?;", id)

	return
}

func GetPostsInReplyTo(id int) (result GetPostsByCriteriaResult) {
	return GetPostsByCriteria("WHERE replyto = ?;", id)
}

func SubmitPost(username string, topic interface{}, subject string, content string, replyto interface{}) (result *GenericResult) {
	var topicID int
	switch v := topic.(type) {
	case string:
		j := GetSectionIDByName(topic.(string))
		if j.Error != nil {
			result.Error = fmt.Errorf("Couldn't get info for the %v secion; %v", topic, j.Error.Error())
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

	fmt.Println(replyID)

	result = new(GenericResult)

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
	userid_ := GetUserIDByName(username)
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
	result.Result, err = execResult.LastInsertId()
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
