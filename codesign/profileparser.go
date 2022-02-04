package codesign

import (
	"bytes"
	"crypto/sha1"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pkcs12"

	"github.com/fullsailor/pkcs7"
	plist "howett.net/plist"
)

//ProfileAndCertificate contains a profiles raw bytes,
//a parsed MobileProvisioningProfile struct to access the fields,
//the p12 sha1 fingerprint, x509.Certificate and the raw p12 bytes
//belonging to this profile.
type ProfileAndCertificate struct {
	RawData                   []byte
	MobileProvisioningProfile MobileProvisioningProfile
	CertificateSha1           string
	SigningCert               *x509.Certificate
	P12Bytes                  []byte
}

//MobileProvisioningProfile is an exact representation of a *.mobileprovision plist
type MobileProvisioningProfile struct {
	AppIDName                   string
	ApplicationIdentifierPrefix []string
	CreationDate                time.Time
	Platform                    []string
	IsXcodeManaged              bool
	DeveloperCertificates       [][]byte
	Entitlements                map[string]interface{}
	ExpirationDate              time.Time
	Name                        string
	ProvisionedDevices          []string
	TeamIdentifier              []string
	TeamName                    string
	TimeToLive                  int
	UUID                        string
	Version                     int
}

//FindProfileForDevice finds the correct profile for a given device udid out of an array
//of profiles and returns the index of the correct profile or -1 if the device is not in any of them
func FindProfileForDevice(udid string, profileAndCertificates []ProfileAndCertificate) int {
	for profileIndex, profileAndCertificate := range profileAndCertificates {
		for _, profileUdid := range profileAndCertificate.MobileProvisioningProfile.ProvisionedDevices {
			if profileUdid == udid {
				return profileIndex
			}
		}
	}
	return -1
}

func verifyP12CertIsInProfile(p12cert *x509.Certificate, certificates []*x509.Certificate) bool {
	if len(certificates) == 0 {
		return false
	}
	p12certHash := getSha1Fingerprint(p12cert)
	for _, cert := range certificates {
		if p12certHash == getSha1Fingerprint(cert) {
			return true
		}
	}
	return false
}

//IsEnterpriseProfile returns true if there is an enterprise profile at profilePath and
//false otherwise.
func IsEnterpriseProfile(profilePath string) bool {
	profileBytes, err := ioutil.ReadFile(profilePath)
	if err != nil {
		return false
	}
	p7, err := pkcs7.Parse(profileBytes)
	if err != nil {
		return false
	}

	decoder := plist.NewDecoder(bytes.NewReader(p7.Content))

	var profile map[string]interface{}
	err = decoder.Decode(&profile)
	if err != nil {
		return false
	}
	if val, ok := profile["ProvisionsAllDevices"]; ok {
		return val.(bool)
	}
	return false
}

//ParseProfiles looks for *.mobileprovision in the given path and parses each of them.
//It returns an error if the path does not contain any profiles.
func ParseProfiles(profilesPath string, profilePassword string, useSingleCertificate bool) ([]ProfileAndCertificate, error) {
	result := []ProfileAndCertificate{}
	singleCertificatePath := ""
	profiles, err := filepath.Glob(path.Join(profilesPath, "*.mobileprovision"))
	if err != nil {
		return result, err
	}
	if useSingleCertificate {
		//New profiles parsing with a single .p12 and multiple mobileprovision
		var certs []string
		certs, err = filepath.Glob(path.Join(profilesPath, "*.p12"))
		if len(certs) == 0 {
			return result, fmt.Errorf("no .p12 certificate was found in the profile path %s", profilesPath)
		}
		if len(certs) > 1 {
			log.Warn("more than one .p12 was found in the profile path %s . using the first in the list...", profilesPath)
		}
		singleCertificatePath = certs[0]
		log.Infof("using certificate '%s'", singleCertificatePath)
	}
	//Default profiles parsing
	for _, file := range profiles {
		log.Infof("parsing profile '%s'", file)
		var profile ProfileAndCertificate
		if useSingleCertificate {
			profile, err = ParseProfileWithSingleCertificate(singleCertificatePath, file, profilePassword)
		} else {
			profile, err = ParseProfile(file, profilePassword)
		}
		if err != nil {
			return result, err
		}
		result = append(result, profile)
	}
	if len(result) == 0 {
		return result, fmt.Errorf("no profiles found in path %s", profilesPath)
	}
	return result, nil
}

//ParseProfile extracts the plist from a pkcs7 signed mobileprovision file.
//It decodes the plist into a go struct. Additionally, a p12 certificate
//must be present next to the profile with the same filename.
// Example: test.mobileprovision and test.p12 must both be present or the parser will fail.
// The parser also checks if the p12 certificate is contained in the profile to prevent errors.
//It returns a ProfileAndCertificate struct containing everything needed for signing.
func ParseProfile(profilePath string, profilePassword string) (ProfileAndCertificate, error) {
	profileBytes, err := ioutil.ReadFile(profilePath)
	if err != nil {
		return ProfileAndCertificate{}, err
	}
	p12bytes, err := ioutil.ReadFile(strings.Replace(profilePath, ".mobileprovision", ".p12", 1))
	if err != nil {
		return ProfileAndCertificate{}, fmt.Errorf("Failed reading p12 file for %s with err: %+v", profilePath, err)
	}

	_, cert, err := pkcs12.Decode(p12bytes, profilePassword)
	if err != nil {
		return ProfileAndCertificate{}, fmt.Errorf("Failed parsing p12 certificate with: %+v", err)
	}

	p7, err := pkcs7.Parse(profileBytes)
	if err != nil {
		return ProfileAndCertificate{}, err
	}

	decoder := plist.NewDecoder(bytes.NewReader(p7.Content))

	var profile MobileProvisioningProfile
	err = decoder.Decode(&profile)

	parsedDeveloperCertificates := make([]*x509.Certificate, len(profile.DeveloperCertificates))

	for i, certBytes := range profile.DeveloperCertificates {
		cert, err := x509.ParseCertificate(certBytes)
		parsedDeveloperCertificates[i] = cert
		if err != nil {
			return ProfileAndCertificate{}, err
		}
	}

	if !verifyP12CertIsInProfile(cert, parsedDeveloperCertificates) {
		return ProfileAndCertificate{}, fmt.Errorf("p12 certificate is not contained in provisioning profile, wrong profile file for this p12")
	}

	return ProfileAndCertificate{MobileProvisioningProfile: profile,
		RawData:         profileBytes,
		CertificateSha1: getSha1Fingerprint(cert),
		P12Bytes:        p12bytes,
		SigningCert:     cert,
	}, err
}

//ParseProfileWithSingleCertificate is exactly like ParseProfile but instead of mobileprovision/.p12 tuples,
//we give a single .p12 path located in the folder alongside the mobileprovision files
//TODO To refactor. this function should be discarded and instead, changes changes should be applied to ParseProfile. This is mainly done for test compatibility and potential breaking changes
func ParseProfileWithSingleCertificate(certificatePath string, profilePath string, profilePassword string) (ProfileAndCertificate, error) {
	profileBytes, err := ioutil.ReadFile(profilePath)
	if err != nil {
		return ProfileAndCertificate{}, err
	}
	p12bytes, err := ioutil.ReadFile(certificatePath)
	if err != nil {
		return ProfileAndCertificate{}, fmt.Errorf("Failed reading p12 file for %s with err: %+v", profilePath, err)
	}

	_, cert, err := pkcs12.Decode(p12bytes, profilePassword)
	if err != nil {
		return ProfileAndCertificate{}, fmt.Errorf("Failed parsing p12 certificate with: %+v", err)
	}

	p7, err := pkcs7.Parse(profileBytes)
	if err != nil {
		return ProfileAndCertificate{}, err
	}

	decoder := plist.NewDecoder(bytes.NewReader(p7.Content))

	var profile MobileProvisioningProfile
	err = decoder.Decode(&profile)

	parsedDeveloperCertificates := make([]*x509.Certificate, len(profile.DeveloperCertificates))

	for i, certBytes := range profile.DeveloperCertificates {
		cert, err := x509.ParseCertificate(certBytes)
		parsedDeveloperCertificates[i] = cert
		if err != nil {
			return ProfileAndCertificate{}, err
		}
	}

	if !verifyP12CertIsInProfile(cert, parsedDeveloperCertificates) {
		return ProfileAndCertificate{}, fmt.Errorf("p12 certificate is not contained in provisioning profile, wrong profile file for this p12")
	}

	return ProfileAndCertificate{MobileProvisioningProfile: profile,
		RawData:         profileBytes,
		CertificateSha1: getSha1Fingerprint(cert),
		P12Bytes:        p12bytes,
		SigningCert:     cert,
	}, err
}

func getSha1Fingerprint(cert *x509.Certificate) string {
	fp := sha1.Sum(cert.Raw)
	return fmt.Sprintf("%x", fp)
}
