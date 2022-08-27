package database

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/IoIxD/LaffForum/debug"
)


type Post struct {
	ID        		int
	Topic     		int
	Subject   		string
	Contents  		string
	Author    		int
	ReplyTo   		int
	Timestamp 		int

	deleted   		interface{}
	deletedtime 	interface{}
	deletedby 		interface{}

	Error 			error
}

func (p Post) Deleted() bool {
	if p.deleted == nil {
		return false
	} else {
		return (p.deleted.(int64) == 1)
	}
}

func (p Post) DeletedTime() string {
	if p.deletedtime == nil {
		return "0"
	} else {
		return p.deletedtime.(string)
	}
}

func (p Post) DeletedBy() string {
	if p.deletedby == nil {
		return "nobody"
	} else {
		return p.deletedby.(string)
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
			result.Error = debug.PublicFacingError("Error while getting post info;", err)
			return
		}
	case int:
		postID = id.(int)
	default:
		result.Error = fmt.Errorf("Invalid type '%v' given.", v)
	}

	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp, deleted, deletedtime, deletedby from `posts` WHERE id = ? LIMIT 1;")
	if err != nil {
		result.Error = debug.PublicFacingError("Error while getting post info;", err)
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
			&result.deleted,
			&result.deletedtime,
			&result.deletedby); err != nil {
			result.Error = debug.PublicFacingError("Error while getting post info;", err)
		}
	}

	return
}

type GetPostsByCriteriaResult struct {
	Posts []Post
	Error error
}

func GetPostsByCriteria(criteria string, values ...any) (result GetPostsByCriteriaResult) {
	statement, err := database.Prepare("SELECT id, topic, subject, contents, author, replyto, timestamp, deleted, deletedtime, deletedby FROM `posts` " + criteria)
	if err != nil {
		result.Error = fmt.Errorf("Couldn't get posts following the relevant criteria; " + err.Error())
		return
	}
	defer statement.Close()

	rows, err := statement.Query(values...)

	for rows.Next() {
		var id interface{}
		var topic int
		var subject string
		var contents string
		var author int
		var replyto int
		var timestamp int
		var deleted interface{}
		var deletedtime interface{}
		var deletedby interface{}
		if err := rows.Scan(&id, &topic, &subject, &contents, &author, &replyto, &timestamp, &deleted, &deletedtime, &deletedby); err != nil {
			result.Error = debug.PublicFacingError("Error getting posts by section name; ", err)
			return
		}

		result.Posts = append(result.Posts, Post{int(id.(int64)), topic, subject, contents, author, replyto, timestamp, deleted, deletedtime, deletedby, nil})
	}
	return
}

func GetPostsBySectionName(name string) (result GetPostsByCriteriaResult) {
	id_ := GetSectionIDByName(name)
	if id_.Error != nil {
		result.Error = debug.PublicFacingError("Error getting posts by section name; ", id_.Error)
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
		result.Error = debug.PublicFacingError("Error getting posts from user; ", id_.Error)
		return
	}
	id := id_.Result.(int64)

	result = GetPostsByCriteria("WHERE author = ? ORDER BY id DESC LIMIT 25;", id)

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

func GetLastFivePosts() (result GetPostsByCriteriaResult) {
	result_ := GetPostsByCriteria("WHERE replyto = ? ORDER BY id DESC LIMIT 5;", 0)
	if result_.Error != nil {
		result.Error = result_.Error
		return
	}
	result.Posts = result_.Posts
	return
}

func GetReadReplyingTo(username string) (result GetPostsByCriteriaResult) {
	useridr := GetUserIDByName(username)
	if(useridr.Error != nil) {
		result.Error = useridr.Error
		return
	} 
	userid := useridr.Result
	usersPosts := GetPostsFromUser(username)
	if(usersPosts.Error != nil) {
		return usersPosts
	}

	var results []Post
	for _, v := range usersPosts.Posts {
		posts := GetPostsByCriteria("WHERE replyto = ? AND unread = 0 AND author != ? ORDER BY id DESC;", v.ID, userid.(int))
		fmt.Println(posts)
		if posts.Error != nil {
			return posts
		}
		for _, n := range posts.Posts {
			results = append(results, n)
		}
	}
	return GetPostsByCriteriaResult{results,nil}
}

func GetUnreadReplyingTo(username string) (result GetPostsByCriteriaResult) {
	useridr := GetUserIDByName(username)
	if(useridr.Error != nil) {
		result.Error = useridr.Error
		return
	} 
	userid := useridr.Result
	usersPosts := GetPostsFromUser(username)
	if(usersPosts.Error != nil) {
		return usersPosts
	}

	var results []Post
	for _, v := range usersPosts.Posts {
		posts := GetPostsByCriteria("WHERE replyto = ? AND unread = 1 AND author != ? ORDER BY id DESC;", v.ID, userid)
		if posts.Error != nil {
			return posts
		}
		for _, n := range posts.Posts {
			results = append(results, n)
		}
	}
	return GetPostsByCriteriaResult{results,nil}
}

// Mark posts as "read" by a session.
func (session *Session) HasRead(id int) (error) {
	err := session.Verify()
	if(err != nil) {
		return err
	}
	err = ExecuteDirect("UPDATE `posts` SET unread = 0 WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

func (session *Session) DeletePost(id interface{}, deletedBy string) (err error) {
	return session.SetDeleteStatus(id, deletedBy, 1)
}
func (session *Session) RestorePost(id interface{}, deletedBy string) (err error) {
	return session.SetDeleteStatus(id, deletedBy, 0)
}
func (session *Session) SetDeleteStatus(id interface{}, deletedBy string, deleteStatus int) (err error) {
	// Check the "session" that wants to modify this post.
	err = session.Verify()
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

	err = ExecuteDirect("UPDATE `posts` SET deleted = ?, deletedby = ?, deletedtime = ? WHERE id = ?", deleteStatus, deletedBy, time.Now().Unix(), postID)
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

func (session *Session) SubmitPost(topic interface{}, subject string, content string, replyto interface{}) (result *SubmitPostResult) {
	result = new(SubmitPostResult)

	// Check the "session" that submitted this post.
	err := session.Verify()
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
		result.Error = debug.PublicFacingError("", userid_.Error)
		return
	}
	userid := userid_.Result.(int64)

	timestampInt := time.Now().Unix()
	timestamp := fmt.Sprintf("%v", timestampInt)

	// Submit the post with those values and what we got in the function arguments, and return the new post id.
	execResult, err := statement.Exec(userid, topicID, subject, content, timestamp, replyID)
	if err != nil {
		result.Error = debug.PublicFacingError("", err)
		return
	}
	result.Result = GetPostInfo(fmt.Sprint(result.ID))
	result.ID, err = execResult.LastInsertId()
	if err != nil {
		result.Error = debug.PublicFacingError("", err)
		return
	}

	return
}
