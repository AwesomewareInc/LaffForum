package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
)

var testingOnLocalhost bool

var config struct {
	HCaptchaSecret string
}

func init() {
	file, err := os.Open("config.toml")
	if err != nil {
		fmt.Println(err)
		testingOnLocalhost = true
		return
	}

	if _, err := toml.NewDecoder(file).Decode(&config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type VerifyCaptchaResponse struct {
	Success    bool `json:"success"`
	Error      error
	ErrorCodes []string `json:"error-codes"`
}

func VerifyCaptcha(clientResponse string) (result VerifyCaptchaResponse) {
	if testingOnLocalhost {
		result.Success = true
		return
	}

	resp, err := http.PostForm("https://hcaptcha.com/siteverify", url.Values{
		"response": {clientResponse},
		"secret":   {config.HCaptchaSecret},
	})
	if err != nil {
		result.Error = err
		return
	}
	resp_, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err
		return
	}

	json.Unmarshal(resp_, &result)

	if len(result.ErrorCodes) >= 1 {
		result.Error = fmt.Errorf("")
		for _, v := range result.ErrorCodes {
			switch v {
			case "missing-input-response":
				result.Error = fmt.Errorf("%v\n Please solve the captcha.", result.Error.Error())
			case "invalid-input-response":
				result.Error = fmt.Errorf("%v\n Failed to solve captcha.", result.Error.Error())
			case "invalid-or-already-seen-response":
				result.Error = fmt.Errorf("%v\n Captcha must be resolved.", result.Error.Error())
			default:
				result.Error = fmt.Errorf("%v\n %v (see https://docs.hcaptcha.com/#siteverify-error-codes-table) ", result.Error.Error(), v)
			}
		}
	}

	return
}
