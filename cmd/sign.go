package cmd

import (
	"io/ioutil"
	"os"

	"github.com/danielpaulus/app-signer/api"
	"github.com/danielpaulus/app-signer/architecturecheck"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type SigningInputs struct {
	ReferenceUdid              string
	ProfileCertificatePassword string
	ProfilesPath               string
	OutputFileName             string
	IpaFileToSign              string
}

var signingInputs = SigningInputs{}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign your apps or ipa using a list of mobile provision, and without a reference udid",
	Long: `By providing your .p12 certificate as well as a list of mobileprovision files, 
you will be able to sign your app or ipa with every mobileprovision available in the profiles path`,
	PreRun: func(cmd *cobra.Command, args []string) {
		logrusConfiguration()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := sign(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	enableBaseSigningRequirements(signCmd, false)
}

func enableBaseSigningRequirements(cmd *cobra.Command, udidRequired bool) {
	cmd.Flags().StringVar(&signingInputs.ReferenceUdid, "udid", "", "Reference udid in order to select the provisioning profile located in your profiles folder")
	cmd.Flags().StringVar(&signingInputs.ProfileCertificatePassword, "p12password", "", "Password for your .p12 located in the your profiles folder")
	cmd.Flags().StringVar(&signingInputs.ProfilesPath, "profilespath", "", "Path to your profiles folder. It should contains a list of mobileprovision as well as a single .p12 associated with them")
	cmd.Flags().StringVar(&signingInputs.OutputFileName, "output", "", "Output path for the signed app or ipa")
	cmd.Flags().StringVar(&signingInputs.IpaFileToSign, "ipa", "", "Path to the target ipa to be signed")
	if udidRequired {
		cmd.MarkFlagRequired("udid")
	}
	cmd.MarkFlagRequired("p12password")
	cmd.MarkFlagRequired("profilespath")
	cmd.MarkFlagRequired("output")
	cmd.MarkFlagRequired("ipa")
}

func sign() error {
	log.Trace("popo")
	err := architecturecheck.CheckLipo()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("lipo is not installed, make sure xcode is installed or add lipo to /usr/bin")
		return err
	}
	workdir, err := ioutil.TempDir("", "pattern")
	defer os.RemoveAll(workdir)
	s, err := api.PrepareSigningWorkspace(workdir, signingInputs.ProfileCertificatePassword, signingInputs.ProfilesPath)
	defer s.Close()
	_, err = api.ResignIPA(s, signingInputs.ReferenceUdid, signingInputs.IpaFileToSign, signingInputs.OutputFileName)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Infof("resigned:")
	return nil
}
