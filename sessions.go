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
	Sessions.sessions = make(map[string]*Session)
	alphabetOnly = *regexp.MustCompile(`[^A-z0-9]`)
}

type SessionsStruct struct {
	sessions map[string]*Session
	mutex    sync.Mutex
}

var Sessions SessionsStruct

type Session struct {
	values map[string]string
	mutex  sync.Mutex
}

func getSession(r *http.Request) *Session {
	ip := r.RemoteAddr
	ua := r.UserAgent()
	ipOnly := strings.Split(ip, ":")[0]

	if ipOnly[0:3] == "127" || ipOnly[0:3] == "192" {
		ip = r.Header.Get("X-Forwarded-For")
		if ip != "" {
			ipParts := strings.Split(ip, ",")
			ipPartParts := strings.Split(ipParts[1], ":")
			ipOnly = ""
			for _, v := range ipPartParts[0][0:2] {
				ipOnly += string(v)
			}
		}
	}

	identifier := string(alphabetOnly.ReplaceAll([]byte(ipOnly+ua), []byte("")))

	return Sessions.get(identifier)
}

func (s *SessionsStruct) get(id string) *Session {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	result, ok := s.sessions[id]
	if !ok {
		s.sessions[id] = new(Session)
		s.sessions[id].values = make(map[string]string)
		result = s.sessions[id]
	}
	return result
}

func (s *Session) get(key string) string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.values[key]
}

func (s *Session) set(key string, value string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.values[key]
	if !ok {
		s.values = make(map[string]string)
	}
	s.values[key] = value
}
