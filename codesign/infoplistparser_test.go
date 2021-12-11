package codesign_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/danielpaulus/app-signer/codesign"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestIpaFile(t *testing.T) {
	ipa := readBytes("fixtures/wda.ipa")

	readerAt := bytes.NewReader(ipa)
	_, directory, err := codesign.ExtractZip(readerAt, int64(len(ipa)))
	if err != nil {
		log.Fatalf("failed extracting: %+v", err)
	}
	defer os.RemoveAll(directory)
	appdir, _ := codesign.FindAppFolder(directory)
	var expectedBundleId = "com.facebook.WebDriverAgentRunner.xctrunner"

	extractedBundleId, err := codesign.GetBundleIdentifier(appdir)

	assert.NoError(t, err)
	assert.Equal(t, expectedBundleId, extractedBundleId)
}

func TestZipFile(t *testing.T) {
	ipa := readBytes("../architecturecheck/fixtures/simulator-app.zip")

	readerAt := bytes.NewReader(ipa)
	_, directory, err := codesign.ExtractZip(readerAt, int64(len(ipa)))
	if err != nil {
		log.Fatalf("failed extracting: %+v", err)
	}
	defer os.RemoveAll(directory)
	appdir, _ := codesign.FindAppFolderVirtualDevice(directory)
	var expectedBundleId = "d.bla"

	extractedBundleId, err := codesign.GetBundleIdentifier(appdir)

	assert.NoError(t, err)
	assert.Equal(t, expectedBundleId, extractedBundleId)
}
