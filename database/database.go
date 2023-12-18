package database

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	sql, err := os.ReadFile("./tableStructure.sql")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	statements := strings.Split(string(sql), ";")
	for _, v := range statements {
		err = ExecuteDirect(v)
		if err != nil {
			// If-oh. fuck.
			if !strings.Contains(err.Error(), "duplicate") {
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

func ExecuteDirect(query string, args ...any) error {
	_, err := database.Exec(query, args[:]...)
	if err != nil {
		// TODO: find out what the actual error type here is.
		if strings.Contains(query, "DROP COLUMN") && !strings.Contains(err.Error(), "no such column") {
			fmt.Println(err)
			return err
		}
	}
	return nil
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

func ExecuteReturnMany(query string, args []any) ([]interface{}, error) {
	dest := make([]any, 0)
	rows, err := database.Query(query, args[:]...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ok interface{}
		if err := rows.Scan(&ok); err != nil {
			return nil, err
		}
		dest = append(dest, ok)
	}
	return dest, nil
}

type GenericResult struct {
	Result any
	Error  error
}

func SQLSanitize(val string) string {
	return SQLEscape.Replace(val)
}

// thread that searches the database for deleted accounts routinely and deletes them
func DeletedAccountThread() {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			found := make([]string, 0)
			statement, err := database.Prepare("SELECT username, deletedtime from `users` WHERE `deleted` = 1;")
			if err != nil {
				fmt.Println(err)
			}

			rows, err := statement.Query()
			for rows.Next() {
				var username string
				var deletedtime int64
				if err := rows.Scan(&username, &deletedtime); err != nil {
					fmt.Println(err)
				}
				deletedTimeParsed := time.Unix(deletedtime, 0)
				if err != nil {
					fmt.Println(err)
				}
				scheduledForDeletion := deletedTimeParsed.Add(time.Hour * 2190)
				if time.Now().After(scheduledForDeletion) {
					found = append(found, username)
				}
			}
			for _, v := range found {
				err := ExecuteDirect("DELETE FROM `users` WHERE `username` = ?;", v)
				if err != nil {
					fmt.Println(err)
				}
				err = ExecuteDirect("INSERT INTO `reservedNames` (username) VALUES (?);", v)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("deleted %v", v)
			}
			statement.Close()
		}
	}
}
