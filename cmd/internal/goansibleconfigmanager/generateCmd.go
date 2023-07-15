package goansibleconfigmanager

import (
	"archive/tar"
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type tarFile struct {
	Name string
	Body []byte
	Mode int64
}

type PlaybookYaml struct {
	Hosts      string   `yaml:"hosts"`
	Connection string   `yaml:"connection"`
	VarsFiles  []string `yaml:"vars_files"`
	Roles      []string `yaml:"roles"`
}

func getGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   `generate [flags] [host]...`,
		Short: `Generate host config files`,
		Run:   runGenerateCmd,
	}

	return cmd
}

func runGenerateCmd(cmd *cobra.Command, args []string) {
	config := loadConfig(cmd)

	hosts := config.GetExistingHostsOrFatal(args)

	log.Println(`Generating config files in`, config.GeneratedConfigDirPath)

	var err = os.MkdirAll(config.GeneratedConfigDirPath, 0700)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	for hostName, hostConfig := range config.Hosts {
		if len(hosts) > 0 && !hosts[hostName] {
			continue
		}

		log.Println(`Generating config for`, hostName)

		playbook := PlaybookYaml{
			Hosts:      `localhost`,
			Connection: `local`,
			Roles:      []string{`basic`},
		}

		var files []tarFile

		// store host variables
		files = append(files, tarFile{
			Name: `playbook_vars.yml`,
			Body: hostConfig.VariablesFile,
		})

		playbook.VarsFiles = append(playbook.VarsFiles, `playbook_vars.yml`)

		// store global variables
		if len(config.GlobalVars) > 0 {
			files = append(files, tarFile{
				Name: `global_vars.yml`,
				Body: config.GlobalVars,
			})

			playbook.VarsFiles = append(playbook.VarsFiles, `global_vars.yml`)
		}

		// generate hosts file
		files = append(files, tarFile{
			Name: `hosts.ini`,
			Body: []byte("localhost ansible_python_interpreter=auto_silent\n"),
		})

		// generate playbook
		playbook.Roles = append(playbook.Roles, hostConfig.ServiceNames...)

		if hostConfig.HasProjects {
			playbook.Roles = append(playbook.Roles, `projects`)
		}

		playbookWrapper := []PlaybookYaml{playbook}
		playbookFile, _ := yaml.Marshal(playbookWrapper)

		files = append(files, tarFile{
			Name: `playbook.yml`,
			Body: playbookFile,
		})

		// copy roles
		files = append(files, getRoleFiles(playbook.Roles, config.RolesPath)...)

		writeTarFile(hostConfig.TarFilePath, files)
	}
}

func getRoleFiles(roles []string, rolesPath string) []tarFile {
	var files []tarFile

	for _, role := range roles {
		rolePath := fmt.Sprintf(`%s%s`, rolesPath, role)

		_ = filepath.Walk(rolePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatal(err)

				return nil
			}

			relativeName := strings.TrimPrefix(path, rolesPath)

			if info.IsDir() {
				return nil
			}

			fileContent, err := os.ReadFile(path)
			if err != nil {
				log.Fatal(`Failed to read file:`, path)
			}

			files = append(files, tarFile{
				Name: relativeName,
				Mode: int64(info.Mode()),
				Body: fileContent,
			})

			return nil
		})
	}

	return files
}

func writeTarFile(filePath string, files []tarFile) {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}

	tarWriter := tar.NewWriter(file)

	for _, file := range files {
		mode := int64(0600)
		if file.Mode != 0 {
			mode = file.Mode
		}

		header := &tar.Header{
			Name: file.Name,
			Mode: mode,
			Size: int64(len(file.Body)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			log.Fatal(err)
		}

		if _, err := tarWriter.Write(file.Body); err != nil {
			log.Fatal(err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		log.Fatal(err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}
