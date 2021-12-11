package main

import (
	"github.com/danielpaulus/app-signer/api"
	"github.com/danielpaulus/app-signer/architecturecheck"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

func main() {

	log.WithFields(log.Fields{"args": os.Args}).Infof("starting iOS appsigner")

	profilePassword := "a"
	profilesDir := "provisioningprofiles"
	ipaFile := "/Users/danielpaulus/privaterepos/app-signer/codesign/fixtures/wda.ipa"
	udid := "00008101-000A402A0EBA001E"
	outputFileName :="resigned.ipa"
	err := architecturecheck.CheckLipo()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("lipo is not installed, make sure xcode is installed or add lipo to /usr/bin")
		return
	}
	workdir, err := ioutil.TempDir("", "pattern")
	defer os.RemoveAll(workdir)
	s, err := api.PrepareSigningWorkspace(workdir, profilePassword, profilesDir)
	defer s.Close()
	_,err = api.ResignIPA(s, udid, ipaFile, outputFileName)
	if err!=nil{
		log.Error(err)
		return
	}
	log.Infof("resigned:")
}
