package codesign

import (
	"fmt"
	"io/ioutil"
	"path"

	"howett.net/plist"
)

const bundleIdentifierKey = "CFBundleIdentifier"
const infoPlist = "Info.plist"

// GetBundleIdentifier finds the Info.plist and returns the bundleid of an app
func GetBundleIdentifier(binDir string) (string, error) {
	plistBytes, err := ioutil.ReadFile(path.Join(binDir, infoPlist))
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	_, err = plist.Unmarshal(plistBytes, &data)
	if err != nil {
		return "", err
	}
	if val, ok := data[bundleIdentifierKey]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("%s not in Info.plist: %+v", bundleIdentifierKey, data)
}
