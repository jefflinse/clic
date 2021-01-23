package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	gobuilder "github.com/jefflinse/clic/builder/go"
	"github.com/jefflinse/clic/io"
	"github.com/jefflinse/clic/spec"
	gowriter "github.com/jefflinse/clic/writer/go"
	"github.com/spf13/cobra"
)

var rootCmd *cobra.Command

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd = &cobra.Command{
		Use:   "clic",
		Short: "command line interface composer",
		Long: `clic - the command line interface composer

Create CLI applications from YAML or JSON specifications.`,
	}

	// top-level commands
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "build an app from an app spec",
		Long:  `build an app from an app spec`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  build,
	}
	buildCmd.Flags().StringP("output-file", "o", "", "app output file location")
	buildCmd.Flags().BoolP("keep-src", "k", false, "preserve intermediate source files")
	rootCmd.AddCommand(buildCmd)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "generate source files from an app spec",
		Long:  `generate source files from an app spec`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  generate,
	}
	generateCmd.Flags().StringP("output-dir", "o", "./out", "location to write output files")
	rootCmd.AddCommand(generateCmd)

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "validate an app spec",
		Long:  `validate an app spec`,
		Args:  cobra.MinimumNArgs(1),
		Run:   validate,
	}
	rootCmd.AddCommand(validateCmd)
}

func build(cmd *cobra.Command, args []string) error {
	app, err := loadAppSpec(args[0])
	if err != nil {
		return err
	}

	srcDir, err := ioutil.TempDir("", "clic.build.*")
	if err != nil {
		return nil
	}
	defer func() {
		if keep, _ := cmd.Flags().GetBool("keep-src"); keep {
			log.Println("preserving sources at", srcDir)
		} else {
			log.Println("cleaning up", srcDir)
			os.RemoveAll(srcDir)
		}
	}()

	sources, err := gowriter.New(app).WriteFiles(srcDir)
	if err != nil {
		return err
	}

	outputFile := cmd.Flag("output-file").Value.String()
	if outputFile == "" {
		outputFile = app.Name
	}

	if !filepath.IsAbs(outputFile) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}

		outputFile = path.Join(cwd, outputFile)
	}

	b := gobuilder.New(sources)
	built, err := b.Build(outputFile)
	if err != nil {
		return err
	}

	log.Println("built", built.Type, "app to", built.Path)

	return nil
}

func generate(cmd *cobra.Command, args []string) error {
	appSpec, err := loadAppSpec(args[0])
	if err != nil {
		return err
	}

	outputDir := cmd.Flag("output-dir").Value.String()
	if !filepath.IsAbs(outputDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}

		outputDir = path.Join(cwd, outputDir)
	}

	if err := io.CreateDirectory(outputDir, false); err != nil {
		return err
	}

	_, err = gowriter.New(appSpec).WriteFiles(outputDir)
	if err != nil {
		return err
	}

	return nil
}

func validate(cmd *cobra.Command, args []string) {
	if _, err := loadAppSpec(args[0]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadAppSpec(file string) (spec.App, error) {
	log.Println("reading app spec from", file)
	app, err := spec.NewAppFromFile(file)
	if err != nil {
		return spec.App{}, err
	}

	log.Println("validating app spec")
	return app.Validate()
}
