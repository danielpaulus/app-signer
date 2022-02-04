package codesign_test

import (
	"log"
	"testing"
	"time"

	"github.com/danielpaulus/app-signer/codesign"
	"github.com/stretchr/testify/assert"
)

func TestDirWithoutProfiles(t *testing.T) {
	_, err := codesign.ParseProfiles(".", "", false)
	assert.Error(t, err)
}

func TestEnterpriseProfileDetection(t *testing.T) {
	shouldBeFalse := codesign.IsEnterpriseProfile("fixtures/embedded.mobileprovision")
	assert.False(t, shouldBeFalse)
}

func TestParsing(t *testing.T) {
	profileAndCertificates, err := codesign.ParseProfiles("../provisioningprofiles", testProfilePassword, false)
	if err != nil {
		log.Fatalf("failed finding profiles %+v", err)
	}
	assert.Equal(t, 1, len(profileAndCertificates))

	profileAndCertificate := profileAndCertificates[0]
	profile := profileAndCertificate.MobileProvisioningProfile
	if assert.NoError(t, err) {
		assert.Equal(t, time.Date(2022, 12, 11, 20, 11, 20, 0, time.UTC), profile.ExpirationDate)
	}

}

func TestFindDeviceInProfile(t *testing.T) {
	profileAndCertificates, err := codesign.ParseProfiles("../provisioningprofiles", testProfilePassword, false)
	if err != nil {
		log.Fatalf("failed finding profiles %+v", err)
	}
	for profileIndex, profileAndCertificate := range profileAndCertificates {
		for _, udid := range profileAndCertificate.MobileProvisioningProfile.ProvisionedDevices {
			foundIndex := codesign.FindProfileForDevice(udid, profileAndCertificates)
			assert.Equal(t, profileIndex, foundIndex)
		}
	}
	assert.Equal(t, -1, codesign.FindProfileForDevice("not contained", profileAndCertificates))
}
