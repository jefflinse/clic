package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jefflinse/clic/builder"
	"github.com/jefflinse/clic/spec"
	"github.com/jefflinse/clic/writer"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "clic",
	Short: "command line interface composer",
	Long: `clic - the command line interface composer

Create CLI applications from YAML or JSON specifications.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		app, err := spec.NewAppFromFile(args[0])
		if err != nil {
			panic(err)
		}

		log.Println("validating app spec")
		if err := app.Validate(); err != nil {
			panic(err)
		}

		w := writer.NewGo(app)
		srcDir, _ := ioutil.TempDir("", "")
		written, err := w.WriteFiles(srcDir)
		if err != nil {
			panic(err)
		} else {
			log.Println("source files written to", written.Dir)
		}

		b := builder.NewGo(app, written)
		bin, _ := ioutil.TempFile("", app.Name)
		built, err := b.Build(bin)
		if err != nil {
			panic(err)
		} else {
			log.Println(built.Type, "app built as", built.Path)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.clic.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".clic" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".clic")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
