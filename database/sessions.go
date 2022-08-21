package database

import (
	"crypto"
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
)

// Files for creaeting and working with "sessions"

type Session struct {
	pubKey 			string
	Username 		string

	Request 		*http.Request
	ResponseWriter 	http.ResponseWriter

	Error error
}

// Function for creating a new session
func NewSession(r *http.Request, w http.ResponseWriter, username string) (result SessionResult) {
	// create the identifier key
	identifier, err := createIdentifier(w)
	if(err != nil) {
		result.Error = err
		return
	}

	// create the key pair to go along with it.
	privKeyRaw, err := rsa.GenerateKey(rand.Reader, 2048)
	if(err != nil) {
		result.Error = err
		return
	}
	pubKeyRaw := &privKeyRaw.PublicKey

	privKey := PrivKeyToString(privKeyRaw)
	if(err != nil) {
		result.Error = err
		return
	}
	pubKey, err := PubKeyToString(pubKeyRaw)
	if(err != nil) {
		result.Error = err
		return
	}

	err = ExecuteDirect("INSERT INTO `sessions` ('genkey', 'pubkey', 'privkey', 'username', 'timestamp') VALUES (?, ?, ?, ?, ?)",
																				fmt.Sprint(identifier),
																				string(privKey),
																				string(pubKey),
																				username,
																				time.Now().Unix())

	if(err != nil) {
		result.Error = err
		return
	}
	return GetSession(r,w)
}

type SessionResult struct {
	*Session
	Error error
}

// Function for getting a session based on the user's information.
func GetSession(r *http.Request, w http.ResponseWriter) (result SessionResult) {
	result.Session = new(Session)
	id_, err := r.Cookie("Token")
	if(err != nil) {
		result.Session.Username = ""
		result.Session.pubKey = ""
		return
	}
	id := strings.Replace(id_.String(),"Token=","",1)
	var pubKey string
	var username string
	err = ExecuteReturn("SELECT pubkey, username FROM `sessions` WHERE genkey = ?;",[]any{id},&pubKey,&username)
	if(err != nil) {
		result.Error = err
		return
	}
	result.Session.pubKey = pubKey
	result.Session.Username = username
	result.Session.Request = r
	result.Session.ResponseWriter = w
	return
}

// allowed characters in cookie headers
var allowedChars = []rune{
	'0','1','2','3','4','5','6','7','8','9','A','B','C','D','E','F','G','H','I','J','K','L','M','N','O','P','Q','R','S',
	'T','U','V','W','X','Y','Z','a','b','c','d','e','f','g','h','i','j','k','l','m','n','o','p','q','r','s','t','u','v','w','x','y','z',
}

// function for creating an identifier and sending it to the user as a cookie.
func createIdentifier(w http.ResponseWriter) (str string, err error) {
	var token string
	for i := 0; i < 32; i++ {
		randomRange := big.NewInt(int64(len(allowedChars)))
		char, err := rand.Int(rand.Reader,randomRange)
		if(err != nil) {
			return "", err
		}
		token += string(allowedChars[int(char.Int64())])
	}
	http.SetCookie(w,&http.Cookie{Name:"Token",Value:token,Expires:time.Now().Add(time.Hour*8760)})
	return token, nil
}
// Function for a session to check itself against the database
func (session Session) Verify() (error) {
	id_, err := session.Request.Cookie("Token")
	if(err != nil) {
		return err
	}
	id := strings.Replace(id_.String(),"Token=","",1)

	var privKey string
	err = ExecuteReturn("SELECT privkey FROM `sessions` WHERE genkey = ?;",[]any{id},&privKey)
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
		cipher, err := rsa.EncryptPKCS1v15(rand.Reader, &lol, []byte(id))
		if(err != nil) {
			return fmt.Errorf("Couldn't encrypt genKey: %v",err)
		}
		_, err = rsa.DecryptPKCS1v15(nil, &privKeyRaw_, cipher)
		if(err != nil) {
			return fmt.Errorf("Keys (probably) don't match, %v",err)
		}
	}
	if privKeyRaw_, ok := privKeyRaw.(ecdsa.PrivateKey); ok {
		r, s, err := ecdsa.Sign(rand.Reader, &privKeyRaw_, []byte(id))
		if(err != nil) {
			return fmt.Errorf("Couldn't encrypt genKey: %v",err)
		}
		lol := pubKeyRaw.(ecdsa.PublicKey)
		valid := ecdsa.Verify(&lol, []byte(id), r, s)
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
	id_, err := session.Request.Cookie("Token")
	if(err != nil) {
		return err.Error()
	}
	id := strings.Replace(id_.String(),"Token=","",1)
	err = ExecuteDirect("DELETE FROM `sessions` WHERE genkey = ?;",id)
	if(err != nil) {
		return err.Error()
	}
	return ""
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