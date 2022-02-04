package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const AppSignerVersion = "1.1"

var (
	Verbose bool
	Trace   bool
	Nojson  bool
	rootCmd = &cobra.Command{
		Use:   "app-signer",
		Short: "app-signer is a Cli helping you sign your iOS apps or ipa on macOS using the official apple codesign tooling",
		Long: `app-signer is a Cli helping you sign iOS apps or ipa using your own certificates and mobileprovision files.
In the background, it uses the official apple codesign tooling on macOS.
By providing your .p12 certificates and associated mobileprovision files as well as a reference device, 
you will be able to sign your app or ipa
`,
		Version: AppSignerVersion,
		PreRun: func(cmd *cobra.Command, args []string) {
			logrusConfiguration()
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("Starting iOS appsigner")
			if err := sign(); err != nil {
				log.Error(err)
			}
		},
	}
)

func init() {
	//Global flags
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable Debug Logging")
	rootCmd.PersistentFlags().BoolVarP(&Trace, "trace", "t", false, "Enable Trace Logging (dump every message)")
	rootCmd.PersistentFlags().BoolVarP(&Nojson, "nojson", "j", false, "Disable JSON output (default)")
	//rootCmd requirements
	enableBaseSigningRequirements(rootCmd, true)
	//Additional commands
	rootCmd.AddCommand(signCmd)
}

func logrusConfiguration() {
	level := ""
	if Verbose {
		level = "debug"
		parsedLevel, err := log.ParseLevel(level)
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(parsedLevel)
	}
	if Trace {
		level = "trace"
		parsedLevel, err := log.ParseLevel(level)
		if err != nil {
			log.Fatal(err)
		}
		log.SetLevel(parsedLevel)
	}
	// TODO We are keeping the old flag behavior but it does not match with the flag meaning. We should change it at some point but it will bring a breaking change to the cli default output
	if Nojson {
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func Execute() error {
	return rootCmd.Execute()
}
