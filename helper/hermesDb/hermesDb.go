package hermesDb

import (
	"context"
	"errors"
	"github.com/jackc/pgx"
	"hermes/helper/hermesVars"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

type ImapConfigs struct {
	Server string
	Port   string
}

type MyDB struct {
	dbName    string
	userName  string
	password  string
	myConnect *pgx.Conn
}

func NewDbConnection(dbName, username, password string) *MyDB {
	connect := &MyDB{}
	connect.dbName = dbName
	connect.userName = username
	connect.password = password
	return connect
}

func (dbCon *MyDB) Connect() error {
	connStr := "user=" + dbCon.userName + " password=" + dbCon.password + " dbname=" + dbCon.dbName
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Println("Connect: " + err.Error())
		return err
	}
	dbCon.myConnect = conn
	return nil
}

func (dbCon *MyDB) CloseConnection() {
	if err := dbCon.myConnect.Close(context.Background()); err != nil {
		log.Println("CloseConnection: " + err.Error())
		return
	}
	return
}

func (dbCon *MyDB) GetEmails(whatEmail string) ([]hermesVars.Email, error, []string) {
	var sqlText string
	var rows pgx.Rows
	var err error
	if whatEmail == "all" {
		sqlText = "SELECT id, email, password, last_scan_dt, owner, priority, groups, is_new, comment, is_checked, status FROM " +
			hermesVars.DbTableEmails
		rows, err = dbCon.myConnect.Query(context.Background(), sqlText)
	} else {
		sqlText = "SELECT id, email, password, last_scan_dt, owner, priority, groups, is_new, comment, is_checked, status FROM " +
			hermesVars.DbTableEmails + " WHERE is_checked=$1 and status=$2"
		rows, err = dbCon.myConnect.Query(context.Background(), sqlText, "0", "1")
	}
	if err != nil {
		log.Println("GetEmails: " + err.Error())
		return nil, err, nil
		//helper.Logging("main", "!!!!!!!!!!   Can't request emails: "+err.Error())
		//panic("")
	}
	var tmpEmail hermesVars.Email
	var tmpEmails []hermesVars.Email
	var tmpID int
	for rows.Next() {
		err = rows.Scan(&tmpID, &tmpEmail.Email, &tmpEmail.Password, &tmpEmail.LastDate, &tmpEmail.Owner, &tmpEmail.Priority,
			&tmpEmail.Groups, &tmpEmail.IsNew, &tmpEmail.Comment, &tmpEmail.IsChecked, &tmpEmail.Status)
		if err != nil {
			log.Println("GetEmails. rows.Next(): " + err.Error())
			//helper.Logging("sub", "Error: "+err.Error())
			continue
		}
		tmpEmail.Id = strconv.Itoa(tmpID)
		tmpEmails = append(tmpEmails, tmpEmail)
	}
	rows.Close()
	var emails []hermesVars.Email
	var logs []string
	for _, eml := range tmpEmails {
		imapName := strings.Split(eml.Email, "@")[1]
		err = dbCon.myConnect.QueryRow(context.Background(), "SELECT imap_host, imap_port FROM "+
			hermesVars.DbTableEmailsCfg+" WHERE name=$1", imapName).Scan(&eml.ImapServ, &eml.ImapPort)
		if err != nil {
			log.Println("GetEmails. GetImap Configs. rows.Next(): " + err.Error())
			logs = append(logs, err.Error()+" for "+eml.Email)
			continue
		}
		emails = append(emails, eml)
	}
	return emails, nil, logs
}

func (dbCon *MyDB) BlockEmails() error {
	sqlText := "UPDATE " + hermesVars.DbTableEmails + " SET status='process' WHERE is_checked=$1 and status=$2 "
	commandTag, err := dbCon.myConnect.Exec(context.Background(), sqlText, "0", "1")
	if err != nil {
		log.Println("UpdateEmail: " + err.Error())
		return err
	}
	if commandTag.RowsAffected() == 0 {
		log.Println("UpdateEmail. commandTag.RowsAffected() == 0: " + err.Error())
		return errors.New("no row found to change last scan date")
	}
	return nil
}

func (dbCon *MyDB) UpdateEmail(eml hermesVars.Email) error {
	sqlText := "UPDATE " + hermesVars.DbTableEmails + " SET email=$1, password=$2, last_scan_dt=$3, status=$4, is_checked=$5, priority=$6, " +
		"comment=$7, is_new=$8 WHERE id=$9"
	commandTag, err := dbCon.myConnect.Exec(context.Background(),
		sqlText,
		eml.Email, eml.Password, eml.LastDate, eml.Status, eml.IsChecked, eml.Priority, eml.Comment, eml.IsNew, eml.Id)
	if err != nil {
		log.Println("UpdateEmail: " + err.Error())
		return err
	}
	if commandTag.RowsAffected() == 0 {
		log.Println("UpdateEmail. commandTag.RowsAffected() == 0: " + err.Error())
		return errors.New("no row found to change last scan date")
	}
	return nil
}

func (dbCon *MyDB) AddArchToBase(archPath string, myEmail hermesVars.Email) error {
	sqlText := "INSERT INTO " + hermesVars.DbTableZips + " (name, path, dwn_date, is_dwn, priority, owner, groups, is_new) VALUES ($1, $2, $3, $4, " +
		"$5, $6, $7, $8)"
	commandTag, err := dbCon.myConnect.Exec(
		context.Background(),
		sqlText,
		filepath.Base(archPath), archPath, myEmail.LastDate, "0", myEmail.Priority, myEmail.Owner, myEmail.Groups, myEmail.IsNew)
	if err != nil {
		log.Println("AddArchToBase: [" + myEmail.Email + ": " + archPath + "] " + err.Error())
		return err
	}
	if commandTag.RowsAffected() != 1 {
		log.Println("AddArchToBase. commandTag.RowsAffected() == 0: " + err.Error())
		return errors.New("can't add archives path to DB")
	}
	return nil
}

func (dbCon *MyDB) InitEmails() error {
	sqlText := "UPDATE " + hermesVars.DbTableEmails + " SET is_checked=$1 WHERE status=$2"
	//commandTag, err := dbCon.myConnect.Exec(context.Background(), sqlText, "0", "1")
	_, err := dbCon.myConnect.Exec(context.Background(), sqlText, "0", "1")
	if err != nil {
		log.Println("InitEmails: " + err.Error())
		return err
	}
	return nil
}

func (dbCon *MyDB) GetCheckingEmails() ([]hermesVars.Email, bool, error) {
	sqlText := "SELECT id, email, password, last_scan_dt, owner, priority, groups, is_new, comment, is_checked, status FROM " +
		hermesVars.DbTableEmails + " WHERE is_checked=$1 and status=$2"
	rows, err := dbCon.myConnect.Query(context.Background(), sqlText, "0", "process")
	if err != nil {
		log.Println("GetCheckingEmails: " + err.Error())
		return nil, false, err
	}
	var emails []hermesVars.Email
	var tmpEmail hermesVars.Email
	needScanCount := 0
	for rows.Next() {
		var tmpId int
		err = rows.Scan(&tmpId, &tmpEmail.Email, &tmpEmail.Password, &tmpEmail.LastDate, &tmpEmail.Owner, &tmpEmail.Priority, &tmpEmail.Groups,
			&tmpEmail.IsNew, &tmpEmail.Comment, &tmpEmail.IsChecked, &tmpEmail.Status)
		if err != nil {
			log.Println("GetCheckingEmails. rows.Next(): " + err.Error())
			continue
		}
		tmpEmail.Id = strconv.Itoa(tmpId)
		emails = append(emails, tmpEmail)
		needScanCount++
	}
	rows.Close()
	sqlText = "SELECT id FROM " + hermesVars.DbTableEmails + " WHERE status=$1"
	rows, err = dbCon.myConnect.Query(context.Background(), sqlText, "blocked")
	if err != nil {
		//helper.Logging("main", "CheckEmails(Count). Can't request emails: "+err.Error())
		log.Println("CheckEmails(Count): " + err.Error())
		return nil, false, err
	}
	defer rows.Close()
	blockedCount := 0
	for rows.Next() {
		blockedCount++
	}
	if needScanCount+blockedCount == 0 {
		return emails, true, nil
	} else {
		return emails, false, nil
	}
}

func (dbCon *MyDB) GetIP() ([]string, error) {
	var ip string
	sqlText := "SELECT value FROM " + hermesVars.DbTableConfigs + " WHERE name=$1"
	err := dbCon.myConnect.QueryRow(context.Background(), sqlText, "ip").Scan(&ip)
	if err != nil {
		log.Println("GetUser: " + err.Error())
		return nil, err
	}
	ips := strings.Split(ip, ",")
	return ips, nil
}

func (dbCon *MyDB) SetScantWorking(value string) error {
	sqlText := "UPDATE " + hermesVars.DbTableConfigs + " SET value=$1 WHERE name=$2"
	commandTag, err := dbCon.myConnect.Exec(context.Background(), sqlText, value, "scan")
	if err != nil {
		log.Println("SetStartWorking: " + err.Error())
		return err
	}
	if commandTag.RowsAffected() == 0 {
		log.Println("SetStartWorking. commandTag.RowsAffected() == 0: " + err.Error())
		return errors.New("no row found to change scan")
	}
	return nil
}

func (dbCon *MyDB) GetScanWorking() (string, error) {
	var value string
	var err error
	sqlText := "SELECT value FROM " + hermesVars.DbTableConfigs + " WHERE name=$1"
	err = dbCon.myConnect.QueryRow(context.Background(), sqlText, "scan").Scan(&value)
	return value, err
}