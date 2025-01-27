package configure

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"io/ioutil"
	"os"
	"path"

	"github.com/manifoldco/promptui"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type CLIVariables struct {
	Variables   []cmdOption
	Responses   map[string]string
	ServiceFile string
	ConfigFile  string
}

// Taken from https://github.com/spf13/cobra/blob/master/doc/yaml_docs.go
type cmdOption struct {
	Name         string
	Shorthand    string
	DefaultValue string
	Usage        string
}

//go:embed templates/*
var templateFiles embed.FS

// Modified from https://github.com/spf13/cobra/blob/master/doc/yaml_docs.go
func genFlagResult(flags *pflag.FlagSet) []cmdOption {
	var result []cmdOption

	flags.VisitAll(func(flag *pflag.Flag) {
		// Todo, when we mark a shorthand is deprecated, but specify an empty message.
		// The flag.ShorthandDeprecated is empty as the shorthand is deprecated.
		// Using len(flag.ShorthandDeprecated) > 0 can't handle this, others are ok.
		if !(len(flag.ShorthandDeprecated) > 0) && len(flag.Shorthand) > 0 {
			opt := cmdOption{
				flag.Name,
				flag.Shorthand,
				flag.DefValue,
				flag.Usage,
			}
			result = append(result, opt)
		} else {
			opt := cmdOption{
				Name:         flag.Name,
				DefaultValue: flag.DefValue,
				Usage:        flag.Usage,
			}
			result = append(result, opt)
		}
	})

	return result
}

func NewPrompter(flags *pflag.FlagSet, configFile, serviceFilePath string) (*CLIVariables, error) {
	setup := CLIVariables{
		Variables: genFlagResult(flags),
	}

	if serviceFilePath == "" {
		serviceFilePath = "/etc/init/"
	}

	serviceFile := "rockbin.conf"

	//determine if upstart or not
	_, err := os.Stat("/sbin/initctl")
	if err != nil {
		serviceFile = "S12Rockbin"
	}

	setup.ConfigFile = configFile
	setup.ServiceFile = path.Join(serviceFilePath, serviceFile)
	// make it executable here.

	return &setup, nil
}

func (p *CLIVariables) PromptUser() error {

	p.Responses = make(map[string]string)

	yamlData, err := ioutil.ReadFile(p.ConfigFile)
	if err != nil {
		fmt.Printf("did not read existing config file: %v", err)
	} else {
		err = yaml.Unmarshal(yamlData, p.Responses)
		if err != nil {
			fmt.Printf("did read existing config file, but failed to parse: %v", err)
		}
	}

	for _, arg := range p.Variables {
		value, ok := p.Responses[arg.Name]
		if !ok {
			value = arg.DefaultValue
		}

		prompt := promptui.Prompt{
			Label:   arg.Usage,
			Default: value,
		}

		answer, err := prompt.Run()

		if err != nil {
			return err
		}

		p.Responses[arg.Name] = answer
	}
	return nil
}

func (p *CLIVariables) WriteOutTemplate(file string, data interface{}) error {
	var (
		outputFile string
		tmplFile   string
		fileMode   fs.FileMode
	)

	fileMode = 0666

	switch file {
	case "config":
		tmplFile = "templates/config.yaml.tmpl"
		outputFile = p.ConfigFile

	case "service":
		tmplFile = path.Join("templates", fmt.Sprintf("%s.%s", path.Base(p.ServiceFile), "tmpl"))
		outputFile = p.ServiceFile
		fileMode = 0755
	}
	fmt.Printf("Writing %s file to: %s\n", file, outputFile)

	t, err := template.ParseFS(templateFiles, tmplFile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Dir(outputFile), 0755); err != nil {
		return err
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = t.Execute(f, data); err != nil {
		return err
	}

	if err := os.Chmod(outputFile, fileMode); err != nil {
		return err
	}

	return nil
}
