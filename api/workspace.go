package api

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/danielpaulus/app-signer/codesign"
	log "github.com/sirupsen/logrus"
	"howett.net/plist"
)

type certAndEntitlement struct {
	certPath        string
	entitlementPath string
	certsha1        string
}

//SigningWorkspace contains the workdir and allows for parsing provisioning profiles.
//It also keeps which certificates are stored where in the workspace dir and knows where the keychain is.
type SigningWorkspace struct {
	workdir         string
	profiles        []codesign.ProfileAndCertificate
	extractedFiles  []certAndEntitlement
	keychainPath    string
	profilePassword string
}

//NewSigningWorkspace set up a new Workspace with a new workdir
func NewSigningWorkspace(workdir string, profilePassword string) SigningWorkspace {
	return SigningWorkspace{workdir: workdir, profilePassword: profilePassword}
}

//PrepareProfiles parses the mobileprovisioning profiles in the given profilesDir.
//It extracts entitlements and stores P12 files, as well associating the correct sha1 fingerprints.
func (s *SigningWorkspace) PrepareProfiles(profilesDir string) error {
	profiles, err := codesign.ParseProfiles(profilesDir, s.profilePassword)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("loading profiles failed")
		return err
	}
	err = os.Mkdir(path.Join(s.workdir, "sign"), 0777)
	if err != nil {
		log.Errorf("failed creating dir in workspace err: %+v", err)
		return err
	}
	err = ioutil.WriteFile(path.Join(s.workdir, "sign", "test.txt"), []byte("some file"), 0777)
	if err != nil {
		log.Errorf("failed creating sample file in workspace sign dir err: %+v", err)
		return err
	}

	log.Infof("found %d profiles", len(profiles))
	s.profiles = profiles
	s.extractedFiles = make([]certAndEntitlement, len(profiles))
	for i, profile := range profiles {
		log.Infof("extracting files for profile: %s", profile.MobileProvisioningProfile.Name)
		bytes, err := plist.Marshal(profile.MobileProvisioningProfile.Entitlements, plist.XMLFormat)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("failed converting to plist")
			return err
		}
		entitlementName := path.Join(s.workdir, profile.MobileProvisioningProfile.Name+"-entitlements.plist")
		log.Infof("extracting entitlements to: '%s'", entitlementName)
		err = ioutil.WriteFile(entitlementName, bytes, 0644)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("writing entitlements failed")
			return err
		}
		certfile := path.Join(s.workdir, profile.MobileProvisioningProfile.Name+"-signingcert.p12")

		log.Infof("extracting signing certificate %s to: '%s'", profile.CertificateSha1, certfile)
		err = ioutil.WriteFile(certfile, profile.P12Bytes, 0644)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("writing certificate failed")
			return err
		}

		s.extractedFiles[i] = certAndEntitlement{certPath: certfile, certsha1: profile.CertificateSha1, entitlementPath: entitlementName}

	}
	return nil
}

//PrepareKeychain creates a new Keychain, unlocks it, disables the timeout
//installs the certificates we found and adds the new keychain to the keychain search list.
func (s *SigningWorkspace) PrepareKeychain(keychainName string) error {
	keychain := path.Join(s.workdir, keychainName)
	err := codesign.CreateKeychain(keychain)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("creating keychain failed")
		return err
	}
	log.Infof("keychain created: %s", keychain)
	err = codesign.UnlockKeychain(keychain)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("unlocking keychain failed")
		return err
	}

	err = codesign.DisableTimeoutForKeychain(keychain)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("disabling timeout keychain failed")
		return err
	}

	for _, cert := range s.extractedFiles {
		log.Infof("installing %s to keychain", cert.certPath)
		err = codesign.AddX509CertificateToKeychain(keychain, cert.certPath, s.profilePassword)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("installing cert failed")
			return err
		}
	}
	err = codesign.AddKeychainToSearchList(keychain)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Errorf("failed adding keychain to searchlist")
		return err
	}

	s.keychainPath = keychain
	return nil
}

//Close removes the keychain that was created from the systems keychain search list
func (s *SigningWorkspace) Close() {
	log.Infof("removing %s from keychain search list", s.keychainPath)
	err := codesign.RemoveFromKeychainSearchList(s.keychainPath)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warn("removing keychain from search list failed")
	}
}

//GetConfig creates codesign.SigningConfig from the workspace's internal data
func (s *SigningWorkspace) GetConfig(index int) codesign.SigningConfig {
	return codesign.SigningConfig{
		CertSha1:             strings.ToUpper(s.extractedFiles[index].certsha1),
		EntitlementsFilePath: s.extractedFiles[index].entitlementPath,
		KeychainPath:         s.keychainPath,
		ProfileBytes:         s.profiles[index].RawData,
	}
}

//TestSigning executes a simple codesign operation to check it works still.
func (s *SigningWorkspace) TestSigning() error {
	length := len(s.profiles)

	for i := 0; i < length; i++ {
		config := s.GetConfig(i)
		cmd := exec.Command("/usr/bin/codesign", "-vv", "--keychain", config.KeychainPath, "--deep", "--force", "--sign", config.CertSha1, path.Join(s.workdir, "sign", "test.txt"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.WithFields(
				log.Fields{"cert": config.CertSha1, "error": err, "cmd": cmd, "output": string(output)}).Infof("codesign test signing failed")
			return err
		}
	}
	return nil
}
