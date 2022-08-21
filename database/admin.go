package database

import (
	"fmt"
	"os"
	"strings"
)


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
