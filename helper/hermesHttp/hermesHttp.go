package hermesHttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"hermes/helper/hermesDb"
	"hermes/helper/hermesVars"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	InProcess = "prcs"
	Ok        = "ok"
)

type Connection struct {
	myUrl string
}

func refactorIPs(ip []string, value string) []string {
	tmpIP := make([]string, len(ip)-1)
	copy(tmpIP, ip[1:])
	tmpIP[len(tmpIP)-1] = tmpIP[len(tmpIP)-1] + value
	return tmpIP
}

func StartDownload(emls *[]hermesVars.Email, spams string, ipList []string) string {
	jsonReq := hermesVars.JsonRequest{
		Extras:   refactorIPs(ipList, "/start"),
		Emails:   *emls,
		SpamList: spams,
	}
	byteJson, _ := json.Marshal(&jsonReq)
	resp, err := http.Post(ipList[0], "application/json", bytes.NewReader(byteJson))
	if err != nil {
		return err.Error()
	}
	if resp == nil {
		return "resp empty"
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Println("StartDownload. resp.Body.Close(): " + err.Error())
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err.Error()
	}
	jsonResp := hermesVars.JsonRequest{}
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return err.Error()
	}
	if jsonResp.Result == "ok" {
		return "ok"
	} else {
		return jsonResp.Result + ": " + jsonResp.Error + " (" + jsonResp.From + ")"
	}
}

func IsFinishDownload(eml hermesVars.Email, ipList []string) (string, []string, string, error) {
	jsonReq := hermesVars.JsonRequest{
		Extras: refactorIPs(ipList, "/check"),
		Emails: []hermesVars.Email{eml},
	}
	byteJson, _ := json.Marshal(&jsonReq)
	response, err := http.Post(ipList[0], "application/json", bytes.NewReader(byteJson))
	if err != nil {
		return "false", nil, "", err
	}
	if response == nil {
		return "false", nil, "", errors.New("resp empty")
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Println("IsFinishDownload. response.Body.Close(): " + err.Error())
		}
	}()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "false", nil, "", err
	}

	resp := hermesVars.JsonRequest{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "false", nil, "", err
	}
	return resp.Result, resp.Emails[0].Archives, resp.Emails[0].LastDate, nil
}

func DownloadEmails(eml hermesVars.Email, ipList []string) string {
	if len(eml.Archives) == 0 {
		return "1"
	}
	emailsPath := hermesVars.EmailsPath + time.Now().Format("02.01.2006") + "/" + eml.Email
	if err := os.MkdirAll(emailsPath, 0777); err != nil {
		log.Println(err)
	}
	dbConnection := hermesDb.NewDbConnection(hermesVars.DbName, hermesVars.DbUsername, hermesVars.DbUserPassword)
	err := dbConnection.Connect()
	if err != nil {
		return err.Error()
	}
	defer dbConnection.CloseConnection()
	i := 0
	for _, archive := range eml.Archives {
		jsonReq := hermesVars.JsonRequest{
			Result: archive,
			Extras: refactorIPs(ipList, "/dwn"),
			Emails: []hermesVars.Email{eml},
		}
		byteJson, _ := json.Marshal(&jsonReq)

		response, err := http.Post(ipList[0], "application/json", bytes.NewReader(byteJson))
		if err != nil {
			continue
		}
		if response == nil {
			continue
		}
		out, err := os.Create(emailsPath + "/" + filepath.Base(archive))
		if err != nil {
			continue
		}
		_, err = io.Copy(out, response.Body)
		if err != nil {
			continue
		}
		dbConnection.AddArchToBase(emailsPath + "/" + filepath.Base(archive), eml)
		out.Close()
		response.Body.Close()
		i++
	}
	if i == 0 {
		return "0"
	}
	if i > 0 && i <= len(eml.Archives) {
		eml.Comment = "downloaded: " + string(i) + "/" + string(len(eml.Archives))
		return "1"
	}
	return "1"
}

func ClearVPS(ipList []string) error {
	jsonReq := hermesVars.JsonRequest{
		Result: "all",
		Extras: refactorIPs(ipList, "/clear"),
	}
	byteJson, _ := json.Marshal(&jsonReq)
	response, err := http.Post(ipList[0], "application/json", bytes.NewReader(byteJson))
	if err != nil {
		log.Println(err)
		return err
	}
	if response == nil {
		return errors.New("ClearVPS. Nil response")
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Print(err)
		}
	}()
	resp := hermesVars.JsonRequest{}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return errors.New("ClearVPS. Can't unmarshal")
	}
	if resp.Result == "1" {
		return nil
	} else {
		return errors.New(resp.Result)
	}
}
