package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Files for creaeting and working with "sessions"

// sessions
type SessionsStruct struct {
	sessions map[string]*Session
	mutex    sync.Mutex
}
type Session struct {
	genKey 		string
	pubKey 		string
	Username 	string

	Error error
}
var sessions SessionsStruct

func init() {
	sessions = SessionsStruct{}
	sessions.sessions = make(map[string]*Session)
	alphabetOnly = *regexp.MustCompile(`[^A-z0-9]`)
}

// Function for creating a new session
func NewSession(r *http.Request, username string) (error) {
	// create the identifier key
	identifier := createIdentifier(r)
	// create the key pair to go along with it.
	privKeyRaw, err := rsa.GenerateKey(rand.Reader, 512)
	if(err != nil) {
		return err
	}
	pubKeyRaw := &privKeyRaw.PublicKey

	privKey := PrivKeyToString(privKeyRaw)
	if(err != nil) {
		return err
	}
	pubKey, err := PubKeyToString(pubKeyRaw)
	if(err != nil) {
		return err
	}

	err = ExecuteDirect("INSERT INTO `sessions` ('genkey', 'pubkey', 'privkey', 'username', 'timestamp') VALUES (?, ?, ?, ?, ?)",
																				identifier,
																				string(privKey),
																				string(pubKey),
																				username,
																				time.Now().Unix())

	if(err != nil) {
		return err
	}
	return nil
}

type SessionResult struct {
	*Session
	Error error
}

// Function for getting a session based on the user's information.
func GetSession(r *http.Request) (result SessionResult) {
	id := createIdentifier(r)
	var pubKey string
	var username string
	err := ExecuteReturn("SELECT pubkey, username FROM `sessions` WHERE genkey = ?;",[]any{id},&pubKey,&username)
	if(err != nil) {
		result.Error = err
		return
	}
	result.Session = new(Session)
	result.Session.genKey = id
	result.Session.pubKey = pubKey
	result.Session.Username = username
	return
}

// regex for stripping a string to letters/numbers only
var alphabetOnly regexp.Regexp

// function for creating an identifier based on the user's permenant section of their IP, and their user agent.
func createIdentifier(r *http.Request) (string) {
	ip := r.RemoteAddr
	ua := r.UserAgent()
	ipOnly := strings.Split(ip, ":")[0]

	if ipOnly[0] != '[' {
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
	}

	return string(alphabetOnly.ReplaceAll([]byte(ipOnly+ua), []byte("")))
}

// Function for a session to check itself against the database
func (session *Session) Verify() (error) {
	var privKey string
	err := ExecuteReturn("SELECT privkey FROM `sessions` WHERE genkey = ?;",[]any{session.genKey},&privKey)
	if(err != nil) {
		return err
	}
	privKeyRaw, err := PrivKeyFromString(privKey)
	if(err != nil) {
		return fmt.Errorf("Error parsing the private key: %v",err)
	}
	pubKeyRaw, err := PubKeyFromString(session.pubKey)
	if(err != nil) {
		return fmt.Errorf("Error parsing the public key: %v",err)
	}
	if privKeyRaw_, ok := privKeyRaw.(rsa.PrivateKey); ok {
		lol := pubKeyRaw.(rsa.PublicKey)
		cipher, err := rsa.EncryptPKCS1v15(rand.Reader, &lol, []byte(session.genKey))
		if(err != nil) {
			return fmt.Errorf("Couldn't encrypt genKey: %v",err)
		}
		_, err = rsa.DecryptPKCS1v15(nil, &privKeyRaw_, cipher)
		if(err != nil) {
			return fmt.Errorf("Keys (probably) don't match, %v",err)
		}
	}
	if privKeyRaw_, ok := privKeyRaw.(ecdsa.PrivateKey); ok {
		r, s, err := ecdsa.Sign(rand.Reader, &privKeyRaw_, []byte(session.genKey))
		if(err != nil) {
			return fmt.Errorf("Couldn't encrypt genKey: %v",err)
		}
		lol := pubKeyRaw.(ecdsa.PublicKey)
		valid := ecdsa.Verify(&lol, []byte(session.genKey), r, s)
		if(!valid) {
			return fmt.Errorf("Keys don't match")
		}
	}
	return nil
}

// Function for getting the user info of a stored username
func (session *Session) Me() UserInfo {
	if session.Username == "" {
		return UserInfo{}
	} else {
		return GetUserInfo(session.Username)
	}
}

// Function for clearing a session, effectively logging out.
func (session *Session) Clear() string {
	err := ExecuteDirect("DELETE FROM `sessions` WHERE genkey = ?;",session.genKey)
	return err.Error()
}

func PrivKeyToString(key *rsa.PrivateKey) (string) {
	return string(pem.EncodeToMemory(
		&pem.Block{
			Type: "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	))
}
func PrivKeyFromString(str string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(str))
	if(block == nil) {
		return nil, fmt.Errorf("Couldn't parse block")
	}

	var key crypto.PrivateKey
	var err1, err2, err3 error

	if key, err1 = x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err2 = x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		switch key_ := key.(type) {
			case rsa.PrivateKey, ecdsa.PrivateKey:
				return &key, nil
			default:
				return nil, fmt.Errorf("found unknown private key type in PKCS#8 wrapping: %v",key_)
		}
	}
	if key, err3 = x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("Couldn't parse private key in any format; \nPKCS1: %v, \nPKCS8: %v, EC: %v",err1,err2,err3)
}
func PubKeyToString(key *rsa.PublicKey) (string, error) {
	str := x509.MarshalPKCS1PublicKey(key)
	if (err != nil) {
		return "", err
	}
	return string(pem.EncodeToMemory(
		&pem.Block{
			Type: "RSA PUBLIC KEY",
			Bytes: str,
		},
	)), nil
}
func PubKeyFromString(str string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(str))
	if(block == nil) {
		return nil, fmt.Errorf("Couldn't parse block")
	}

	var key crypto.PublicKey
	var err1, err2 error

	if key, err1 = x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err2 = x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("Couldn't parse private key in any format; \nPKCS1: %v, EC: %v",err1,err2)
}