package codesign_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/danielpaulus/app-signer/codesign"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

//Compress a bunch of files, uncompress them and make sure
//all the files are still present.
func TestCompressionCode(t *testing.T) {
	//Sets up example folders and files for testing compression
	//Good for making sure all edge cases like empty directories are working fine
	dirToCompress, err := setUpExampleDir()
	if err != nil {
		log.Fatalf("failed setting up temp dir for testing with err %+v", err)
	}
	defer os.RemoveAll(dirToCompress)
	buf := bytes.Buffer{}
	originalFiles, err := codesign.GetFiles(dirToCompress)
	err = codesign.CompressToZip(dirToCompress, &buf)
	assert.NoError(t, err)

	duration, outputTempDir, err := codesign.ExtractZip(bytes.NewReader(buf.Bytes()), int64(len(buf.Bytes())))
	defer os.RemoveAll(outputTempDir)

	log.Infof("took %v to compress %d files", duration, len(originalFiles))

	extractedFiles, err := codesign.GetFiles(outputTempDir)
	assert.Equal(t, len(extractedFiles), len(originalFiles))

	//replace the root folder from the pathnames so we can match files
	for i := 0; i < len(extractedFiles); i++ {
		extractedFiles[i] = strings.Replace(extractedFiles[i], outputTempDir+"/", "", 1)
		originalFiles[i] = strings.Replace(originalFiles[i], dirToCompress+"/", "", 1)
	}

	assert.ElementsMatch(t, originalFiles, extractedFiles)
}

func setUpExampleDir() (string, error) {
	tempdir, err := ioutil.TempDir("", "appsigner-extract-test")
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(path.Join(tempdir, "empty", "dir", "included", "here"), 0777)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(path.Join(tempdir, "test.txt"), []byte("example file"), 777)
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(path.Join(tempdir, ".test.hidden"), []byte("hidden example file"), 777)
	if err != nil {
		return "", err
	}
	return tempdir, nil
}
