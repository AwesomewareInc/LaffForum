package database

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Files for creaeting and working with "sessions"

type Session struct {
	_privKey string
	username string
	keys     []string

	request        *http.Request
	responseWriter http.ResponseWriter

	err error
}

// Function for creating a new session
func NewSession(r *http.Request, w http.ResponseWriter, username string) (result SessionResult) {
	privKeyRaw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		result.Error = err
		return
	}
	pubKeyRaw := &privKeyRaw.PublicKey

	privKey := PrivKeyToString(privKeyRaw)
	if err != nil {
		result.Error = err
		return
	}

	pubKey, err := PubKeyToString(pubKeyRaw)
	if err != nil {
		result.Error = err
		return
	}

	err = ExecuteDirect("INSERT INTO `sessions` ('pubkey', 'username', 'timestamp') VALUES (?, ?, ?)",
		string(pubKey),
		username,
		time.Now().Unix())

	if err != nil {
		result.Error = err
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "PrivKey", Value: privKey, Expires: time.Now().Add(time.Hour * 8760)})
	http.SetCookie(w, &http.Cookie{Name: "Username", Value: username, Expires: time.Now().Add(time.Hour * 8760)})

	return GetSession(r, w)
}

func (s Session) PrivKey() string {
	return s._privKey
}
func (s Session) PrivKeyCookie() (string, error) {
	// We want to check if the user has the legacy "Token" cookie and log them out if so.
	id_, err := s.request.Cookie("Token")
	if err == nil {
		if id_.String() != "Token=" {
			http.SetCookie(s.responseWriter, &http.Cookie{Name: "Token", Value: "", Expires: time.Now().Add(time.Hour * 8760)})
		}
	}
	id_, err = s.request.Cookie("PrivKey")
	if err != nil {
		return "", err
	}
	privKey := strings.Replace(id_.String(), "PrivKey=", "", 1)
	privKey = strings.ReplaceAll(privKey, "\"", "")
	if privKey == "" {
		s.username = ""
		return "", fmt.Errorf("No private key stored.")
	}
	privKey = strings.ReplaceAll(strings.ReplaceAll(privKey, "-----END", "\n-----END"), "KEY-----", "KEY-----\n")
	return privKey, nil
}
func (s Session) Username() string { return s.username }

type SessionResult struct {
	*Session
	Error error
}

// allowed characters in cookie headers
var allowedChars = []rune{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S',
	'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
}

// function for creating an identifier and sending it to the user as a cookie.
func createIdentifier() (str string, err error) {
	var token string
	for i := 0; i < 32; i++ {
		randomRange := big.NewInt(int64(len(allowedChars)))
		char, err := rand.Int(rand.Reader, randomRange)
		if err != nil {
			return "", err
		}
		token += string(allowedChars[int(char.Int64())])
	}
	return token, nil
}

// Function for getting a session based on the user's information.
func GetSession(r *http.Request, w http.ResponseWriter) (result SessionResult) {
	result.Session = new(Session)
	result.Session.request = r
	result.Session.responseWriter = w

	privKey, err := result.Session.PrivKeyCookie()
	if err != nil {
		result.Session.username = ""
		result.Session._privKey = ""
		return
	}

	user_, err := r.Cookie("Username")
	if err != nil {
		result.Session.username = ""
		result.Session._privKey = ""
		return
	}
	user := strings.Replace(user_.String(), "Username=", "", 1)

	keys, err := ExecuteReturnMany("SELECT pubkey FROM `sessions` WHERE username = ?;", []any{&user})
	if err != nil {
		result.Error = err
		return
	}

	result.Session.username = user

	result.Session._privKey = privKey
	result.Session.keys = make([]string, 0)
	for _, key := range keys {
		if k, ok := key.(string); ok {
			result.Session.keys = append(result.Session.keys, k)
		}
	}

	if len(result.Session.keys) <= 0 {
		result.Session.username = ""
		result.Session._privKey = ""
		return
	}
	return
}

func (session Session) GetValidPublicKeys(privKey string) ([]string, error) {
	publicKeys := make([]string, 0)
	privKeyRaw, err := PrivKeyFromString(privKey)
	if err != nil {
		return nil, fmt.Errorf("Error parsing the private key: %v", err)
	}

	id, err := createIdentifier()
	if err != nil {
		return nil, fmt.Errorf("Error creating a cipher: %v", err)
	}

	if privKeyRaw_, ok := privKeyRaw.(*rsa.PrivateKey); ok {
		for _, key := range session.keys {
			pubKeyRaw, err := PubKeyFromString(key)
			if err != nil {
				return nil, fmt.Errorf("Error parsing the public key: %v", err)
			}
			if lol, ok := pubKeyRaw.(*rsa.PublicKey); ok {
				cipher, err := rsa.EncryptPKCS1v15(rand.Reader, lol, []byte(id))
				if err != nil {
					return nil, fmt.Errorf("Couldn't encrypt privKey: %v", err)
				}
				// silently fail if they don't match.
				_, err = rsa.DecryptPKCS1v15(nil, privKeyRaw_, cipher)
				if err == nil {
					publicKeys = append(publicKeys, key)
				}
			} else {
				return nil, fmt.Errorf("Invalid public key. Try clearing your cookies.")
			}

		}
	} else if privKeyRaw_, ok := privKeyRaw.(*ecdsa.PrivateKey); ok {
		for _, key := range session.keys {
			pubKeyRaw, err := PubKeyFromString(key)
			if err != nil {
				return nil, fmt.Errorf("Error parsing the public key: %v", err)
			}
			r, s, err := ecdsa.Sign(rand.Reader, privKeyRaw_, []byte(id))
			if err != nil {
				return nil, fmt.Errorf("Couldn't encrypt privKey: %v", err)
			}
			if lol, ok := pubKeyRaw.(*ecdsa.PublicKey); ok {
				valid := ecdsa.Verify(lol, []byte(privKey), r, s)
				if valid {
					publicKeys = append(publicKeys, key)
				}
			} else {
				return nil, fmt.Errorf("Invalid public key. Try clearing your cookies.")
			}

		}
	} else {
		return nil, fmt.Errorf("Invalid key type. Key was %v", privKey)
	}

	return publicKeys, nil
}

// Function for a session to check itself against the database
func (session Session) Verify() error {
	privKey, err := session.PrivKeyCookie()
	if err != nil {
		return err
	}
	publicKeys, err := session.GetValidPublicKeys(privKey)
	if err != nil {
		return err
	}
	if len(publicKeys) <= 0 {
		session.username = ""
		session._privKey = ""
		return fmt.Errorf("User's private key does not match any public key in the database.")
	}
	return nil
}

// Function for getting the user info of a stored username
func (session *Session) Me() UserInfo {
	if session.username == "" {
		return UserInfo{}
	} else {

		return GetUserInfo(session.Username())
	}
}

// Function for clearing a session, effectively logging out.
func (session *Session) Clear() string {
	privKey, err := session.PrivKeyCookie()
	if err != nil {
		return err.Error()
	}
	publicKeys, err := session.GetValidPublicKeys(privKey)
	if err != nil {
		return err.Error()
	}
	session.username = ""
	session._privKey = ""
	if len(publicKeys) >= 1 {
		for _, key := range publicKeys {
			ExecuteDirect("DELETE FROM `sessions` WHERE pubKey = ?", key)
		}
	}

	http.SetCookie(session.responseWriter, &http.Cookie{Name: "PrivKey", Value: "", Expires: time.Now().Add(time.Hour * 8760)})
	http.SetCookie(session.responseWriter, &http.Cookie{Name: "Username", Value: "", Expires: time.Now().Add(time.Hour * 8760)})

	return ""
}

func PrivKeyToString(key *rsa.PrivateKey) string {
	return string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	))
}
func PrivKeyFromString(str string) (crypto.PrivateKey, error) {
	block, wh := pem.Decode([]byte(str))
	if block == nil {
		return nil, fmt.Errorf("Couldn't parse block.\nThe private key is below (don't worry about it being printed, this is specific to the site, just don't share it)\n%v", string(wh))
	}

	var key crypto.PrivateKey
	var err1, err2, err3 error

	if key, err1 = x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err2 = x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		switch key_ := key.(type) {
		case rsa.PrivateKey, ecdsa.PrivateKey, dsa.PrivateKey:
			return &key, nil
		default:
			return nil, fmt.Errorf("found unknown private key type in PKCS#8 wrapping: %v", key_)
		}
	}
	if key, err3 = x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("Couldn't parse private key in any format; \nPKCS1: %v, \nPKCS8: %v, EC: %v", err1, err2, err3)
}
func PubKeyToString(key *rsa.PublicKey) (string, error) {
	str := x509.MarshalPKCS1PublicKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: str,
		},
	)), nil
}
func PubKeyFromString(str string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(str))
	if block == nil {
		return nil, fmt.Errorf("Couldn't parse block")
	}

	var key crypto.PublicKey
	var err1, err2, err3 error

	if key, err1 = x509.ParsePKCS1PublicKey(block.Bytes); err1 == nil {
		return key, nil
	}
	if key, err2 = x509.ParsePKIXPublicKey(block.Bytes); err2 == nil {
		return key, nil
	}
	if key, err3 = ssh.ParsePublicKey(block.Bytes); err3 == nil {
		return key, nil
	}

	return nil, fmt.Errorf("Couldn't parse public key in any format; \n\nPKCS1: %v\n\nPKIX: %v\n\nSSH: %v", err1, err2, err3)
}
