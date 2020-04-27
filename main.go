package main

import (
	"hermes/helper"
	"hermes/helper/hermesVars"
	"log"
)

func main() {
	if helper.IsScanWorking() {
		return
	}
	log.Println("Start scanning...")
	defer log.Println("Scan emails finished!")
	helper.Logging(hermesVars.LogFile, "main", " ")
	helper.Logging(hermesVars.LogFile, "main", "------------------------------ START ------------------------------")
	defer helper.Logging(hermesVars.LogFile, "main", "------------------------------ FINISH ------------------------------")
	if err := helper.SetScanWorking("1"); err != nil {
		helper.Logging(hermesVars.LogFile, "main", "Can't set start scan: " + err.Error())
	}
	defer func() {
		if err := helper.SetScanWorking("0"); err != nil {
			helper.Logging(hermesVars.LogFile, "main", "Can't set start scan: " + err.Error())
		}
	}()
	helper.ScanEmails()

}