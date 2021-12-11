package architecturecheck_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/danielpaulus/app-signer/architecturecheck"
	"github.com/danielpaulus/app-signer/codesign"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPhysicalDevice(t *testing.T) {
	ipa := readBytes("fixtures/wda.ipa")

	readerAt := bytes.NewReader(ipa)
	_, directory, err := codesign.ExtractZip(readerAt, int64(len(ipa)))
	if err != nil {
		log.Fatalf("failed extracting: %+v", err)
	}
	defer os.RemoveAll(directory)
	appdir, _ := codesign.FindAppFolder(directory)

	extractedArchs, err := architecturecheck.ExtractArchitectures(appdir)

	assert.NoError(t, err)
	assert.NotContains(t, extractedArchs, "x86_64")
	assert.False(t, architecturecheck.IsSimulatorApp(extractedArchs))
}

func TestVirtualDevice(t *testing.T) {
	ipa := readBytes("fixtures/simulator-app.zip")

	readerAt := bytes.NewReader(ipa)
	_, directory, err := codesign.ExtractZip(readerAt, int64(len(ipa)))
	if err != nil {
		log.Fatalf("failed extracting: %+v", err)
	}
	defer os.RemoveAll(directory)
	appdir, _ := FindAppFolderVirtualDevice(directory)

	extractedArchs, err := architecturecheck.ExtractArchitectures(appdir)

	assert.NoError(t, err)
	assert.Contains(t, extractedArchs, "x86_64")
	assert.True(t, architecturecheck.IsSimulatorApp(extractedArchs))
}

func TestLipoCheck(t *testing.T) {
	assert.NoError(t, architecturecheck.CheckLipo())
}

//FindAppFolderVirtualDevice returns the path of the *.app directory
//which must be in the root of the unzipped file.
//or an error if there is no .app directory or more than one.
func FindAppFolderVirtualDevice(rootDir string) (string, error) {
	appFolders, err := filepath.Glob(path.Join(rootDir, "*.app"))
	if err != nil {
		return "", err
	}
	if len(appFolders) != 1 {
		return "", fmt.Errorf("found more or less than exactly one app folder: %+v", appFolders)
	}
	return appFolders[0], nil
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
