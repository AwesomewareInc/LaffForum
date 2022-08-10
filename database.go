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
	"github.com/pelletier/go-toml/v2"
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
	// database settings
	f, err := os.Open("config.toml")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = toml.NewDecoder(f).Decode(&DatabaseConfig)
	if err != nil {
		fmt.Println(err)
		return
	}

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
	_, err = database.Exec(query, args...)
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
	Error error
}

func PublicFacingError(msg string, err error) (error) {
	pc, filename_, line, _ := runtime.Caller(1)

	filenameParts := strings.Split(filename_,"/")
	filename := filenameParts[len(filenameParts)-1]

	funcname_ := runtime.FuncForPC(pc).Name()
	funcnames := strings.Split(funcname_,".")
	funcname := funcnames[len(funcnames)-1]

	stacktrace := string(debug.Stack())

	return fmt.Errorf("%v at %v:%v in %v(), %v. \n %v",msg,filename,line,funcname,err.Error(),stacktrace)
}

// ==============
// USER SHIT
// ==============

func UserExists(username string) string {
	var id int
	err := ExecuteReturn("SELECT count(id) from `users` WHERE username = ?;", []interface{}{username}, &id)
	if err != nil {
		return "Couldn't validate username; "+err.Error()
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
		return "Couldn't generate password; "+err.Error()
	}
	statement, err := database.Prepare("INSERT INTO `users` (username, password, prettyname, timestamp) VALUES (?, ?, ?, ?);")
	if err != nil {
		return "Couldn't prepare statement to create user; "+err.Error()
	}
	defer statement.Close()
	_, err = statement.Exec(username, hashedPass, prettyname, fmt.Sprintf("%v", time.Now().Unix()))
	if err != nil {
		return "Couldn't create user; "+err.Error()
	}

	return ""
}

func GetUserIDByName(name string) (result GenericResult) {
	result.Result = -1
	err := ExecuteReturn("SELECT count(id) from `users` WHERE username = ?;", []interface{}{name}, &result.Result)
	if err != nil {
		result.Error = PublicFacingError("Error while getting user id by name;",err)
		return
	}
	return
}

func GetUsernameByID(id int) (result GenericResult) {
	err := ExecuteReturn("SELECT username from `users` WHERE id = ?;", []interface{}{id}, &result.Result)
	if err != nil {
		result.Error = PublicFacingError("Error while getting username by id;",err)
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
	ID   int
	Name string
}

// Todo: GetSectionsResult
func GetSections() []Section {
	statement, err := database.Prepare("SELECT id, name from `sections`;")
	if err != nil {
		return []Section{{-1, err.Error()}}
	}
	defer statement.Close()

	rows, err := statement.Query()
	results := make([]Section, 0)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			results = append(results, Section{-1, err.Error()})
		}
		results = append(results, Section{id, name})
	}
	return results
}

func GetSectionIDByName(name string) (result GenericResult) {
	err := ExecuteReturn("SELECT id from `sections` WHERE name = ?;",[]interface{}{name},&result.Result)
	if err != nil {
		result.Error = PublicFacingError("Error while getting section id by name;",err)
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
}

type GetPostInfoResult struct {
	Post
	Error error
}

func GetPostInfo(id_ string) (result GetPostInfoResult) {
	id, err := strconv.Atoi(id_)
	if err != nil {
		result.Error = PublicFacingError("Error while getting post info;",err)
		return
	}

	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp from `posts` WHERE id = ? LIMIT 1;")
	if err != nil {
		result.Error = PublicFacingError("Error while getting post info;",err)
		return
	}
	defer statement.Close()

	rows, err := statement.Query(id)
	var postID int
	var topic int
	var subject string
	var contents string
	var author int
	var replyto int
	var timestamp int
	rows.Next()
	if err := rows.Scan(&postID, &topic, &subject, &contents, &author, &replyto, &timestamp); err != nil {
		result.Error = PublicFacingError("Error while getting post info;",err)
	} else {
		result.Post = Post{postID, topic, subject, contents, author, replyto, timestamp}
	}
	return
}

type GetPostsBySectionNameResult struct {
	Posts []Post
	Error error
}

func GetPostsBySectionName(name string) (result GetPostsBySectionNameResult) {
	result.Posts = make([]Post,0)
	id_ := GetSectionIDByName(name)
	if(id_.Error != nil) {
		result.Error = PublicFacingError("Error getting posts by section name; ",id_.Error)
		return
	}
	id := id_.Result.(int64)

	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp from posts WHERE `topic` = ?;")
	if err != nil {
		result.Error = err
		return 
	}
	defer statement.Close()

	rows, err := statement.Query(id)
	for rows.Next() {
		var id int

		var topic int
		var subject string
		var contents string
		var author int
		var replyto int
		var timestamp int
		if err := rows.Scan(&id, &topic, &subject, &contents, &author, &replyto, &timestamp); err != nil {
			result.Error = PublicFacingError("Error getting posts by section name; ",err)
			return
		}
		result.Posts = append(result.Posts, Post{id, topic, subject, contents, author, replyto, timestamp})
	}
	return
}

func SubmitPost(username, topic, subject, content string) (result *GenericResult) {
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
	statement, err := database.Prepare("INSERT INTO `posts` (id, author, topic, subject, contents, timestamp, replyto) VALUES (?, ?, ?, ?, ?, ?, ?);")
	if err != nil {
		result.Error = err
		return
	}
	defer statement.Close()

	// Get the necessary values.
	userid_ := GetUserIDByName(username)
	if(userid_.Error != nil) {
		result.Error = PublicFacingError("",userid_.Error)
		return
	}
	userid := userid_.Result.(int64)

	topicid_ := GetSectionIDByName(topic)
	if(topicid_.Error != nil) {
		result.Error = PublicFacingError("",topicid_.Error)
		return
	}
	topicid := topicid_.Result.(int64)

	timestampInt := time.Now().Unix()
	timestamp := fmt.Sprintf("%v", timestampInt)

	// Submit the post with those values and what we got in the function arguments, and return the new post id.
	rows, err := statement.Query(timestampInt+userid+topicid, userid, topicid, subject, content, timestamp, 0)
	result.Result = ""
	rows.Next()
	if err := rows.Scan(&result.Result); err != nil {
		result.Error = PublicFacingError("",err)
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
		err = MakeAdmin(args[1], args[2])
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
	return ExecuteDirect("UPDATE `users` WHERE username = ? SET adminonly = 1;", args)
}
