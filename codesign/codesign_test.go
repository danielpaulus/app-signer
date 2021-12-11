package codesign_test

import (
	"bytes"
	b64 "encoding/base64"
	"github.com/danielpaulus/app-signer/api"
	"github.com/danielpaulus/app-signer/codesign"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.Println("Do stuff BEFORE the tests!")
	p12b64, present := os.LookupEnv("p12")
	if present {
		uDec, _ := b64.URLEncoding.DecodeString(p12b64)
		err := os.WriteFile("../provisioningprofiles/test.p12", uDec, 0644)
		if err != nil {
			log.Error(err)
		}
	}
	profileb64, present := os.LookupEnv("profile")
	if present {
		uDec, _ := b64.URLEncoding.DecodeString(profileb64)
		err := os.WriteFile("../provisioningprofiles/test.mobileprovision", uDec, 0644)
		if err != nil {
			log.Error(err)
		}
	}
	exitVal := m.Run()
	log.Println("Do stuff AFTER the tests!")

	os.Exit(exitVal)
}

const testProfilePassword = "a"

//TestCodeSign tests the resigning process end to end.
//The ipa will be extracted, signed, zipped and in case
//the environment variable udid is specified, installed to a device.
func TestCodeSign(t *testing.T) {

	ipa := readBytes("fixtures/wda.ipa")

	workspace, cleanup := makeWorkspace()
	defer cleanup()

	readerAt := bytes.NewReader(ipa)
	duration, directory, err := codesign.ExtractZip(readerAt, int64(len(ipa)))
	if err != nil {
		log.Fatalf("failed extracting: %+v", err)
	}
	log.Infof("Extraction took:%v", duration)
	defer os.RemoveAll(directory)

	index := 0
	if udid, yes := runOnRealDevice(); yes {
		index = findProfile(udid)
	}
	signingConfig := workspace.GetConfig(index)

	startSigning := time.Now()
	err = codesign.Sign(directory, signingConfig)
	assert.NoError(t, err)
	durationSigning := time.Since(startSigning)
	log.Infof("signing took: %v", durationSigning)

	b := &bytes.Buffer{}

	assert.NoError(t, codesign.Verify(path.Join(directory, "Payload", "WebDriverAgentRunner-Runner.app")))

	compressStart := time.Now()
	err = codesign.CompressToZip(directory, b)
	if err != nil {
		log.Fatalf("Compression failed with %+v", err)
	}
	compressDuration := time.Since(compressStart)
	log.Printf("compressiontook: %v", compressDuration)

	if udid, yes := runOnRealDevice(); yes {
		installOnRealDevice(udid, b.Bytes())
	} else {
		log.Warn("No UDID provided, not running installation on actual device")
	}
}

func runOnRealDevice() (string, bool) {
	udid := os.Getenv("udid")
	udid = "00008101-000A402A0EBA001E"
	return udid, udid != ""
}

func readBytes(name string) []byte {
	file, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func makeWorkspace() (api.SigningWorkspace, func()) {
	dir, err := ioutil.TempDir("", "appsigner-api-test")
	if err != nil {
		log.Fatal(err)
	}

	workspace := api.NewSigningWorkspace(dir, testProfilePassword)
	workspace.PrepareProfiles("../provisioningprofiles")
	workspace.PrepareKeychain("test.keychain")

	cleanUp := func() {
		defer os.RemoveAll(dir)
		defer workspace.Close()
	}
	return workspace, cleanUp
}

func findProfile(udid string) int {
	profiles, err := codesign.ParseProfiles("../provisioningprofiles", testProfilePassword)
	if err != nil {
		log.Fatalf("could not parse profiles %+v", err)
	}
	index := codesign.FindProfileForDevice(udid, profiles)
	if index == -1 {
		log.Fatalf("Device: %s is not in profiles", udid)
	}
	return index
}

func installOnRealDevice(udid string, ipa []byte) {
	ipafile, err := ioutil.TempFile("", "myname-*.ipa")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(ipafile.Name())

	ipafile.Write(ipa)
	ipafile.Close()

	installerlogs, err := exec.Command("ios", "install", "--path", ipafile.Name(), "--udid", udid).CombinedOutput()
	if err != nil {
		log.Fatalf("failed installing, logs: %s with err %+v", string(installerlogs), err)
	}
	log.Info("Install successful")
}
