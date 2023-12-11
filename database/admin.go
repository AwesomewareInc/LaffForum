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
	case "help":
		fmt.Println("createsection, renamesection, deletesection, mkadmin, revadmin, exit")
		return nil
	case "getsections":
		sections := GetSections()
		for _, section := range sections.Results {
			fmt.Printf("%d: %s (admin only: %t)\n", section.ID, section.Name, section.AdminOnly == 1)
		}
	case "createsection":
		if len(args) < 3 {
			return fmt.Errorf("createsection <string sectionname> <int adminonly>")
		}
		err = CreateSection(args[1], args[2])
	case "renamesection":
		if len(args) < 3 {
			return fmt.Errorf("renamesection <string sectionname> <string newname>")
		}
		err = RenameSection(args[2], args[1])
	case "deletesection":
		if len(args) < 2 {
			return fmt.Errorf("deletesection <string sectionname>")
		}
		err = DeleteSection(args[1])
	case "mkadmin":
		if len(args) < 2 {
			return fmt.Errorf("mkadmin <string username>")
		}
		err = MakeAdmin(args[1])
	case "revadmin":
		if len(args) < 2 {
			return fmt.Errorf("mkadmin <string username>")
		}
		err = RemoveAdmin(args[1])
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

func RenameSection(args ...any) error {
	return ExecuteDirect("UPDATE `sections` SET name = ? WHERE name = ?", args...)
}

func DeleteSection(args ...any) error {
	return ExecuteDirect("DELETE FROM `sections` WHERE name = ?;", args...)
}

func MakeAdmin(args ...any) error {
	return ExecuteDirect("UPDATE `users` SET admin = 1 WHERE username = ?;", args...)
}

func RemoveAdmin(args ...any) error {
	return ExecuteDirect("UPDATE `users` SET admin = 0 WHERE username = ?;", args...)
}
