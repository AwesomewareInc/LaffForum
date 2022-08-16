package main

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
)

var alphabetOnly regexp.Regexp

func init() {
	Sessions = SessionsStruct{}
	alphabetOnly = *regexp.MustCompile(`[^A-z0-9]`)
}

type SessionsStruct struct {
	sessions 	map[string]*Session
	mutex 		sync.Mutex
}
var Sessions SessionsStruct

type Session struct {
	values 		map[string]string
	mutex 		sync.Mutex
}

func GetSession(r *http.Request) (*Session) {
	ip := r.RemoteAddr
	ua := r.UserAgent()
	ipOnly := strings.Split(ip,":")[0]

	identifier := string(alphabetOnly.ReplaceAll([]byte(ipOnly+ua),[]byte("")))

	return Sessions.Get(identifier)
}

func (s *SessionsStruct) Get(id string) (*Session) {
	s.mutex.Lock()
	result := s.sessions[id]
	/*if(result == nil) {
		s.sessions[id] = new(Session)
		result = s.sessions[id]
	}*/
	s.mutex.Unlock()
	return result
}

func (s *Session) Get(key string) (*string) {
	s.mutex.Lock()
	result := s.values[key]
	s.mutex.Unlock()
	return &result
}

func (s *Session) Set(key string, value string) {
	s.mutex.Lock()
	s.values[key] = value
	s.mutex.Unlock()
}