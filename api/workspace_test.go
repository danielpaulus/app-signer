package api_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/danielpaulus/app-signer/api"
	log "github.com/sirupsen/logrus"
)

func TestWorkspaceInit(t *testing.T) {
	_, _, cleanUp := makeWorkspaceWithoutProfiles()
	defer cleanUp()

}

func makeWorkspaceWithoutProfiles() (api.SigningWorkspace, string, func()) {
	dir, err := ioutil.TempDir("", "resigner-test")
	if err != nil {
		log.Fatal(err)
	}

	workspace := api.NewSigningWorkspace(dir, "")

	cleanUp := func() {
		defer os.RemoveAll(dir)
	}
	return workspace, dir, cleanUp
}
