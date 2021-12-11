package main

import (
	"fmt"
	"github.com/danielpaulus/app-signer/api"
	"github.com/danielpaulus/app-signer/architecturecheck"
	"github.com/docopt/docopt-go"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

func main() {
	version := "1.0"
	usage := fmt.Sprintf(`sign %s

Usage:
  sign --udid=<udid> --p12password=<p12password> --profilespath=<profilespath> --ipa=<ipa> --output=<output> [options]

Options:
  -v --verbose   Enable Debug Logging.
  -t --trace     Enable Trace Logging (dump every message).
  --nojson       Disable JSON output (default).
  -h --help      Show this screen.

The commands work as following:
  `, version)
	arguments, err := docopt.ParseDoc(usage)
	log.WithFields(log.Fields{"args": os.Args}).Infof("starting iOS appsigner")
	udid, _ := arguments.String("--udid")
	profilePassword, _ := arguments.String("--p12password")
	profilespath, _ := arguments.String("--profilespath")
	outputFileName, _ := arguments.String("--output")
	ipaFile, _ := arguments.String("--ipa")

	err = architecturecheck.CheckLipo()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("lipo is not installed, make sure xcode is installed or add lipo to /usr/bin")
		return
	}
	workdir, err := ioutil.TempDir("", "pattern")
	defer os.RemoveAll(workdir)
	s, err := api.PrepareSigningWorkspace(workdir, profilePassword, profilespath)
	defer s.Close()
	_, err = api.ResignIPA(s, udid, ipaFile, outputFileName)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("resigned:")
}
