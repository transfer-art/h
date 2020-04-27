package helper

import (
	"errors"
	"hermes/helper/hermesDb"
	"hermes/helper/hermesHttp"
	"hermes/helper/hermesVars"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

func Logging(logFile, msgType, msg string) {
	logFile = "Logs/" + logFile
	if msgType == "sub" {
		msg = "    " + msg
	}
	msg = "[" + time.Now().Format("02.01.2006 15:04") + "]" + ":     " + msg
	if err := os.MkdirAll(path.Dir(logFile), 0777); err != nil {
		log.Print("os.MkdirAll: ", err)
	}
	var fl *os.File
	var err error
	fl, err = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fl, err = os.Create(logFile)
		if err != nil {
			return
		}
		defer func() {
			if err := fl.Close(); err != nil {
				log.Println("fl.Close[1]: ", err)
			}
		}()
		if _, err = fl.WriteString(msg + "\n"); err != nil {
			log.Println("fl.WriteString[1]: ", err)
		}
		return
	}
	defer func() {
		if err := fl.Close(); err != nil {
			log.Println("fl.Close[2]: ", err)
		}
	}()
	if _, err = fl.WriteString(msg + "\n"); err != nil {
		log.Print("fl.WriteString[2]: ", err)
	}
}

func ScanEmails() {
	ips := getIPs()
	if err := hermesHttp.ClearVPS(ips); err != nil {
		Logging(hermesVars.LogFile, "sub", err.Error())
		log.Print("helper. ClearVPS: " + err.Error())
	}
	dbConnection := hermesDb.NewDbConnection(hermesVars.DbName, hermesVars.DbUsername, hermesVars.DbUserPassword)
	err := dbConnection.Connect()
	if err != nil {
		Logging(hermesVars.LogFile, "main", "Can't connect to DB: "+err.Error())
		return
	}
	defer dbConnection.CloseConnection()
	err = dbConnection.InitEmails()
	if err != nil {
		Logging(hermesVars.LogFile, "main", "InitEmails: "+err.Error())
		return
	}
	emails, err, logs := dbConnection.GetEmails("only works")
	if err != nil {
		Logging(hermesVars.LogFile, "main", "GetEmails: "+err.Error())
		return
	}
	if logs != nil {
		for _, myLog := range logs {
			Logging(hermesVars.LogFile, "main", "GetEmails. Get Imap configs: "+myLog)
		}
	}
	if err = dbConnection.BlockEmails(); err != nil {
		Logging(hermesVars.LogFile, "main", "Block emails: " + err.Error())
	}
	spamFile, err := ioutil.ReadFile("spam_domens")
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create("spam_domens")
			if err != nil {
				Logging(hermesVars.LogFile, "sub", "Spam filter: "+err.Error())
			}
			file.Close()
		} else {
			Logging(hermesVars.LogFile, "sub", "Spam filter: "+err.Error())
		}
	}
	spamList := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(string(spamFile), "\n", " "), "\r", ""))

	if res := hermesHttp.StartDownload(&emails, spamList, ips); res != "ok" {
		Logging(hermesVars.LogFile, "main", "StartDownload: "+res)
		panic(errors.New("StartDownload: " + res))
	}
	//time.Sleep(time.Minute)
	var routineCount int
	var wg sync.WaitGroup
	for {
		emails, isNeedFinish, err := dbConnection.GetCheckingEmails()
		if err != nil {
			Logging(hermesVars.LogFile, "main", "GetCheckingEmails: "+err.Error())
		}
		if isNeedFinish {
			break
		}
		if len(emails) < 5 {
			routineCount = len(emails)
		} else {
			routineCount = 5
		}
		chEmails := make(chan hermesVars.Email, routineCount)
		for i := 0; i < routineCount; i++ {
			wg.Add(1)
			go checkEmail(&wg, chEmails, ips)
		}
		for _, email := range emails {
			chEmails <- email
		}
		time.Sleep(10 * time.Second)
		close(chEmails)
		wg.Wait()
		time.Sleep(time.Minute)
	}
	if err := hermesHttp.ClearVPS(ips); err != nil {
		Logging(hermesVars.LogFile, "sub", err.Error())
		log.Print("helper. ClearVPS: " + err.Error())
	}
}

func checkEmail(wg *sync.WaitGroup, ch chan hermesVars.Email, ips []string) {
	routineDbConnection := hermesDb.NewDbConnection(hermesVars.DbName, hermesVars.DbUsername, hermesVars.DbUserPassword)
	err := routineDbConnection.Connect()
	if err != nil {
		Logging(hermesVars.LogFile, "sub", "routineDbConnection: "+err.Error())
		return
	}
	defer routineDbConnection.CloseConnection()
	defer wg.Done()

	for {
		eml, isOk := <-ch
		if !isOk {
			break
		}
		var res, lastScanDate string
		res, eml.Archives, lastScanDate, err = hermesHttp.IsFinishDownload(eml, ips)
		if err != nil {
			Logging(hermesVars.LogFile, "main", "Check IsFinishDownload: "+err.Error())
			eml.Status = "2"
			eml.Comment = err.Error()
			if err := routineDbConnection.UpdateEmail(eml); err != nil {
				Logging(hermesVars.LogFile, "sub", "UpdateEmail for "+eml.Email+": "+err.Error())
				log.Println(err)
			}
			continue
		}
		switch res {
		case hermesHttp.Ok:
			dwnRes := hermesHttp.DownloadEmails(eml, ips)
			if dwnRes == "1" {
				eml.Status = "1"
				eml.LastDate = lastScanDate
			} else {
				Logging(hermesVars.LogFile, "main", "Main DownloadEmails: "+res)
				eml.Comment = "can't download archives. Error: " + dwnRes
				eml.Status = "2"
			}
			break
		case hermesHttp.InProcess:
			continue
		default:
			eml.Comment = res
			eml.Status = "2"
		}
		eml.IsChecked = "1"
		eml.IsNew = "0"
		if err := routineDbConnection.UpdateEmail(eml); err != nil {
			Logging(hermesVars.LogFile, "sub", "UpdateEmail for "+eml.Email+": "+err.Error())
			log.Println(err)
		}
	}
}

func getIPs() []string {
	dbConnection := hermesDb.NewDbConnection(hermesVars.DbName, hermesVars.DbUsername, hermesVars.DbUserPassword)
	err := dbConnection.Connect()
	if err != nil {
		Logging(hermesVars.LogFile, "main", "EditEmail. Can't connect to DB: "+err.Error())
		return nil
	}
	defer dbConnection.CloseConnection()
	ips, err := dbConnection.GetIP()
	if err != nil {
		panic("Don't have IPs")
	}
	return ips
}

func SetScanWorking(value string) error{
	dbConnection := hermesDb.NewDbConnection(hermesVars.DbName, hermesVars.DbUsername, hermesVars.DbUserPassword)
	err := dbConnection.Connect()
	if err != nil {
		Logging(hermesVars.LogFile, "main", "EditEmail. Can't connect to DB: "+err.Error())
		return err
	}
	defer dbConnection.CloseConnection()
	return dbConnection.SetScantWorking(value)
}

func IsScanWorking() bool {
	dbConnection := hermesDb.NewDbConnection(hermesVars.DbName, hermesVars.DbUsername, hermesVars.DbUserPassword)
	err := dbConnection.Connect()
	if err != nil {
		Logging(hermesVars.LogFile, "main", "IsScanWorking. Can't connect to DB: "+err.Error())
		return true
	}
	defer dbConnection.CloseConnection()
	val, err := dbConnection.GetScanWorking()
	if err != nil {
		Logging(hermesVars.LogFile, "main", "IsScanWorking. GetScanWorking: "+err.Error())
		return true
	}
	if val == "1" {
		return true
	} else {
		return false
	}
}

