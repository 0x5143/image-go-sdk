/**********************************************************************************************
 #
 # Github : github.com/tencentyun/go-sdk
 # File name : sign.go
 # Description : qcloud sign
 #
**********************************************************************************************/
package sign

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const HMAC_LENGTH = 20

type picUrlField struct {
	domain string
	appid  uint
	userid uint
	fileid string
}

func parsePicUrl(url string) (fields picUrlField, e error) {
	params := strings.Split(strings.TrimPrefix(url, "http://"), "/")
	if len(params) < 4 {
		desc := fmt.Sprintf("url format error, url=%s", url)
		e = errors.New(desc)
		return
	}
	fields.domain = params[0]
	var value int
	value, _ = strconv.Atoi(params[1])
	fields.appid = uint(value)
	value, _ = strconv.Atoi(params[2])
	fields.userid = uint(value)
	fields.fileid = params[3]
	e = nil
	return
}

func SignBase(appid uint, secretId string, secretKey string, expire uint, userid uint, url string) (string, error) {
	if "" == secretId || "" == secretKey {
		return "", errors.New("invalid params, secret id or key is empty")
	}

	var fileid string
	if "" != url {
		fields, err := parsePicUrl(url)
		if nil != err {
			return "", err
		}

		fileid = fields.fileid
	}

	now := time.Now().Unix()
	r := rand.New(rand.NewSource(time.Now().Unix()))
	rdm := r.Int31()
	expireTime := expire
	if 0 != expireTime {
		expireTime += uint(now)
	}

	plainStr := fmt.Sprintf("a=%d&k=%s&e=%d&t=%d&r=%d&u=%d&f=%s",
		appid,
		secretId,
		expireTime,
		now,
		rdm,
		userid,
		fileid)

	cryptoStr := []byte(plainStr)
	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write(cryptoStr)
	hmacStr := h.Sum(nil)
	cryptoStr = append(hmacStr, cryptoStr...)
	sign := base64.StdEncoding.EncodeToString(cryptoStr)
	return sign, nil
}

func AppSign(appid uint, secretId string, secretKey string, expire uint, userid uint) (string, error) {
	return SignBase(appid, secretId, secretKey, expire, userid, "")
}

func AppSignOnce(appid uint, secretId string, secretKey string, userid uint, url string) (string, error) {
	return SignBase(appid, secretId, secretKey, 0, userid, url)
}

func Decode(sign string, appid uint, secretId string, secretKey string) (userid uint, expire uint, fileid string, e error) {
	if "" == sign {
		e = errors.New("invalid sign string")
		return 
	}

	cryptoStr, e := base64.StdEncoding.DecodeString(sign)
	if nil != e {
		return
	}else if len(cryptoStr) <= HMAC_LENGTH {
		e = errors.New("sign is too short")
		return
	}

	hmacStr := cryptoStr[0:HMAC_LENGTH]
	cryptoStr = cryptoStr[HMAC_LENGTH:]
	
	//check hmac str
	h := hmac.New(sha1.New, []byte(secretKey))
	h.Write(cryptoStr)
	hmacStr2 := h.Sum(nil)
	if len(hmacStr) != len(hmacStr2) {
		desc := fmt.Sprintf("hmac check failed, hmac1=%s, hmac2=%s", hmacStr, hmacStr2)
		e = errors.New(desc)
		return
	}

	for i:= range hmacStr {
		if hmacStr[i] != hmacStr2[i] {
			desc := fmt.Sprintf("hmac check failed, hmac1=%s, hmac2=%s", hmacStr, hmacStr2)
			e = errors.New(desc)
			return
		}
	}

	//check cryto string
	fields := strings.Split(string(cryptoStr), "&")
	//check appid
	if fields[0] != ("a="+strconv.Itoa(int(appid))) {
		desc := fmt.Sprintf("invalid appid, appid=%d, sign=%s", appid, fields[0])
		e = errors.New(desc)
		return
	}
	//check skey
	if fields[1] != ("k="+secretId) {
		desc := fmt.Sprintf("invalid secret_id, sid=%s, sign=%s", secretId, fields[1])
		e = errors.New(desc)
		return
	}
	//check time
	//[3] is create time
	//[2] is expire time
	tmp, e := strconv.Atoi(strings.TrimLeft(fields[2], "e="))
	if nil != e {
		return
	}
	expire = uint(tmp)
	//check userid
	tmp, e = strconv.Atoi(strings.TrimLeft(fields[5], "u="))
	if nil != e {
		return
	}
	userid = uint(tmp)
	//check fileid
	fileid = strings.TrimLeft(fields[6], "f=")
	/////
	if ( expire == 0 && fileid == "") ||
			(expire != 0 && fileid != "" ) {
		desc := fmt.Sprintf("invalid expire time or fileid, expire=%s, fileid=%s", fields[2], fields[6])
		e = errors.New(desc)
		return
	}

	e = nil
	return
}
