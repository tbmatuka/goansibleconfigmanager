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
	VarFiles   []string `yaml:"var_files"`
	Roles      []string `yaml:"roles"`
}

func getGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Short generate desc",
		Long:  `Long generate description`,
		Run: func(cmd *cobra.Command, args []string) {
			config := loadConfig(cmd)

			log.Println(`Generating config files in`, config.GeneratedConfigDirPath)

			var err = os.MkdirAll(config.GeneratedConfigDirPath, 0700)
			if err != nil && !os.IsExist(err) {
				log.Fatal(err)
			}

			for hostName, hostConfig := range config.Hosts {
				log.Println(`Generating config for`, hostName)

				tarPath := fmt.Sprintf(`%s/%s.tar`, strings.TrimRight(config.GeneratedConfigDirPath, `/`), hostName)

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

				playbook.VarFiles = append(playbook.VarFiles, `playbook_vars.yml`)

				// store global variables
				if len(config.GlobalVars) > 0 {
					files = append(files, tarFile{
						Name: `global_vars.yml`,
						Body: config.GlobalVars,
					})

					playbook.VarFiles = append(playbook.VarFiles, `global_vars.yml`)
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
				for _, role := range playbook.Roles {
					rolePath := fmt.Sprintf(`%s%s`, config.RolesPath, role)

					_ = filepath.Walk(rolePath, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							log.Fatal(err)

							return nil
						}

						relativeName := strings.TrimPrefix(path, config.RolesPath)

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

				writeTarFile(tarPath, files)
			}
		},
	}

	return cmd
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
