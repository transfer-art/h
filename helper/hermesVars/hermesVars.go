package hermesVars

const (
	DbName           = "at_db"    // atlant_db
	DbUsername       = "at_admin" // atlant
	DbUserPassword   = "atTA!#@2" // talant13@
	DbTableEmails    = "atlant_emails"
	DbTableEmailsCfg = "atlant_emails_cfg"
	DbTableZips      = "atlant_zips"
	DbTableConfigs   = "atlant_configs"

	LogFile    = "scan_log.txt"
	EmailsPath = "/home/yel/go/src/hermes/Emails/"
)

type Email struct {
	Id          string   `json:"mid, omitempty"`
	Email       string   `json:"email, omitempty"`
	Password    string   `json:"password, omitempty"`
	LastDate    string   `json:"last_date, omitempty"`
	Owner       string   `json:"owner, omitempty"`
	Priority    string   `json:"priority, omitempty"`
	Groups      string   `json:"groups, omitempty"`
	IsNew       string   `json:"is_new, omitempty"`
	Comment     string   `json:"comment, omitempty"`
	IsChecked   string   `json:"is_checked, omitempty"`
	Description string   `json:"description, omitempty"`
	Status      string   `json:"status, omitempty"`
	Archives    []string `json:"archives, omitempty"`
	ImapServ    string   `json:"imapServer, omitempty"`
	ImapPort    string   `json:"imapPort, omitempty"`
}

type JsonRequest struct {
	Result   string   `json:"result, omitempty"`
	Error    string   `json:"error, omitempty"`
	From     string   `json:"from, omitempty"`
	SpamList string   `json:"spam_list, omitempty"`
	Extras   []string `json:"extras, omitempty"`
	Emails   []Email  `json:"emails, omitempty"`
}
