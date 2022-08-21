package database

import (
	"fmt"

	"github.com/IoIxD/LaffForum/debug"
)

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
		if finalInt, ok := j.Result.(int64); ok {
			sectionID = int(finalInt)
		} else {
			sectionID = -1
		}
		
	case int:
		sectionID = id.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
		return
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
		result.Error = debug.PublicFacingError("Error while getting section id by name;", err)
		return
	}
	return
}

func GetSectionNameByID(id int) (result GenericResult) {
	err := ExecuteReturn("SELECT name from `sections` WHERE id = ?;", []interface{}{id}, &result.Result)
	if err != nil {
		result.Error = debug.PublicFacingError("Error while getting section name by id;", err)
		return
	}
	return
}
