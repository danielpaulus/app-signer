package api

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/danielpaulus/app-signer/architecturecheck"
	"github.com/danielpaulus/app-signer/codesign"
)

func PrepareSigningWorkspace(workdir string, profilePassword string, profilesDir string, useSingleCertificate bool) (SigningWorkspace, error) {
	workDirPath, err := filepath.Abs(workdir)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("workdir path Abs failed")
		return SigningWorkspace{}, err
	}
	log.Info("cleaning workdir")
	err = os.RemoveAll(workDirPath)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("removing workdir failed")
		return SigningWorkspace{}, err
	}
	err = os.Mkdir(workDirPath, 0777)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("removing workdir failed")
		return SigningWorkspace{}, err
	}

	signingWorkspace := NewSigningWorkspace(workDirPath, profilePassword)
	if useSingleCertificate {
		signingWorkspace.EnableSingleCertificateUsage()
	}
	err = signingWorkspace.PrepareProfiles(profilesDir)
	if err != nil {
		log.Error("appsigner failed to start")
		return SigningWorkspace{}, err
	}

	err = signingWorkspace.PrepareKeychain("appsigner.keychain")
	if err != nil {
		log.Error("appsigner failed to start")
		return SigningWorkspace{}, err
	}
	err = signingWorkspace.TestSigning()
	if err != nil {
		log.Error("test signing failed", err)
	}
	return signingWorkspace, nil
}

func ResignIPA(s SigningWorkspace, udid string, ipafilePath string, outputFileOrFolder string) (string, error) {
	if udid == "" && !s.useSingleCertificate {
		return "", fmt.Errorf("udid was empty")
	}
	ipafile, err := os.Open(ipafilePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %s with err: %v", ipafilePath, err)
	}
	info, err := ipafile.Stat()
	if err != nil {
		return "", fmt.Errorf("failed getting file info for %+v err: %v", ipafile, err)
	}

	_, directory, err := codesign.ExtractZip(ipafile, info.Size())
	if err != nil {
		return "", fmt.Errorf("failed extracting ipafile")
	}
	defer os.RemoveAll(directory)

	if codesign.ContainsAppstoreApp(directory) {
		log.Warn("this is a appstore build, are you sure it should be resigned?")
	}

	appFolder, err := codesign.FindAppFolder(directory)
	if err != nil {
		fmt.Errorf("could not find .app folder in extracted ipa payload folder")
	}

	//if the appstore build check succeeds, the app is guaranteed to have an embedded.mobileprovision profile
	if codesign.IsEnterpriseProfile(path.Join(appFolder, "embedded.mobileprovision")) {
		log.Warn("this app was signed with an enterprise certificate, resigning makes no sense")
	}

	archs, err := architecturecheck.ExtractArchitectures(appFolder)
	if err != nil {
		return "", fmt.Errorf("could not determine build architecture of build, run 'lipo -info appDir/appExecutable' to debug %+v", err)
	}
	if architecturecheck.IsSimulatorApp(archs) {
		return "", fmt.Errorf("invalid build architectures: %v, was this build for a simulator?", archs)
	}
	if s.useSingleCertificate {
		//By going here, we assume that outputFileOrFolder is a path to a folder and not a complete filepath
		//For each profile available and configured in the list s.profiles, we will sign the same app multiple times
		//and create multiple signed app in the target folder
		for index, profile := range s.profiles {
			profileConfig := s.GetConfig(index)
			appName, appExt := getAppDetails(ipafile.Name())
			outputFile := fmt.Sprintf("%s/%s_signed_%s%s", path.Clean(outputFileOrFolder), appName, profile.MobileProvisioningProfile.Name, appExt)
			log.Debugf("full output filepath: %s", outputFile)
			_, err = resignIPA(directory, profileConfig, outputFile)
			if err != nil {
				log.Errorf("an error occured during the signing process: %s", err)
			}
		}
		return "", nil
	}
	//Here we keep the main way of signing an app using a reference udid and tuples of .p12/mobileprovision
	//Expected output should be a single app signed
	index := codesign.FindProfileForDevice(udid, s.profiles)
	if index == -1 {
		return "", fmt.Errorf("the device '%s' is not contained in any profile", udid)
	}
	profileConfig := s.GetConfig(index)
	return resignIPA(directory, profileConfig, outputFileOrFolder)
}

func getAppDetails(appPath string) (string, string) {
	baseFilePath := path.Base(appPath)
	appExtension := path.Ext(baseFilePath)
	return strings.TrimSuffix(baseFilePath, appExtension), appExtension
}

func resignIPA(directory string, config codesign.SigningConfig, outputFile string) (string, error) {
	err := codesign.Sign(directory, config)
	if err != nil {
		return "", fmt.Errorf("failed signing app: %v", err)
	}
	f, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	err = codesign.CompressToZip(directory, f)
	if err != nil {
		return "", fmt.Errorf("failed zipping app: %v", err)
	}
	log.Info("succeeded signing")
	log.Info(outputFile)
	return outputFile, nil
}
