package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

const (
	pubLen int = 12
)

func checkOTP(w http.ResponseWriter, r *http.Request, dal *Dal) {
	if r.URL.Query()["otp"] == nil || r.URL.Query()["nonce"] == nil || r.URL.Query()["id"] == nil {
		reply(w, "", "", MISSING_PARAMETER, "", dal)
		return
	}
	otp := r.URL.Query()["otp"][0]
	nonce := r.URL.Query()["nonce"][0]
	id := r.URL.Query()["id"][0]
	if len(otp) < pubLen {
		reply(w, otp, nonce, BAD_OTP, id, dal)
		return
	}
	pub := otp[:pubLen]

	k, err := dal.GetKey(pub)
	if err != nil {
		reply(w, otp, nonce, BAD_OTP, id, dal)
		return
	} else {
		k, err = Gate(k, otp)
		if err != nil {
			reply(w, otp, nonce, err.Error(), id, dal)
			return
		} else {
			err = dal.UpdateKey(k)
			if err != nil {
				log.Println("fail to update key counter/session")
				return
			}
			reply(w, otp, nonce, OK, id, dal)
			return
		}
	}
}

func Sign(values []string, key string) string {
	payload := ""
	for _, v := range values {
		payload += v + "&"
	}
	payload = payload[:len(payload)-1]

	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(payload))
	s := mac.Sum(nil)

	return base64.StdEncoding.EncodeToString(s)
}

func loadKey(id string, dal *Dal) (string, error) {
	i, err := dal.GetApp(id)
	if err != nil {
		return "", errors.New(NO_SUCH_CLIENT)
	}

	return *i, nil
}

func reply(w http.ResponseWriter, otp, nonce, status, id string, dal *Dal) {
	values := []string{}
	key := ""
	err := errors.New("")

	values = append(values, "nonce="+nonce)
	values = append(values, "opt="+otp)
	if status != MISSING_PARAMETER {
		key, err = loadKey(id, dal)
		if err == nil {
			values = append(values, "status="+status)
		} else {
			values = append(values, "status="+err.Error())
		}
	} else {
		values = append(values, "status="+status)
	}
	values = append(values, "t="+time.Now().Format(time.RFC3339))
	if status != MISSING_PARAMETER {
		values = append(values, "h="+Sign(values, key))
	}

	ret := ""
	for _, v := range values {
		ret += v + "\n"
	}

	w.Write([]byte(ret))
}

func runAPI(dal *Dal, port string) {
	r := mux.NewRouter()

	r.HandleFunc("/wsapi/2.0/verify", func(w http.ResponseWriter, r *http.Request) {
		checkOTP(w, r, dal)
	}).Methods("GET")

	http.Handle("/", r)
	log.Println("Listening on port " + port + "...")
	http.ListenAndServe(":"+port, nil)
}
