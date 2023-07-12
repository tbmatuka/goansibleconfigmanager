package goansibleconfigmanager

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Listen struct {
		Host    string
		Port    uint16
		SSLCert string
		SSLKey  string
	}

	// used for generating URLs
	URLPrefix string

	// Generate configs in this dir and serve them from here
	GeneratedConfigDirPath string

	// get roles from this dir when generating a config
	RolesPath string

	// configs for each of the hosts
	Hosts map[string]HostConfig

	// Global env variables that will be added to each playbook
	GlobalVars []byte
}

type YamlConfig struct {
	Listen struct {
		Host    string `yaml:"host"`
		Port    uint16 `yaml:"port"`
		SSLCert string `yaml:"ssl_cert"`
		SSLKey  string `yaml:"ssl_key"`
	}

	URLPrefix string `yaml:"url_prefix"`

	GeneratedConfigDirPath string `yaml:"generated_config_dir_path"`

	GlobalVars map[string]interface{} `yaml:"global_vars"`
}

type HostConfig struct {
	Name          string
	Key           string
	ServiceNames  []string
	HasProjects   bool
	VariablesFile []byte
}

type HostYamlConfig struct {
	ConfigKey string                 `yaml:"config_key"`
	Services  map[string]interface{} `yaml:"services"`
	Projects  map[string]interface{} `yaml:"projects"`
}

func loadConfig(cmd *cobra.Command) Config {
	configDir := fmt.Sprintf(`%s/`, strings.TrimRight(cmd.Flag(`config-dir`).Value.String(), `/`))

	configFilePath := fmt.Sprintf(`%s%s`, configDir, `ansible-config-manager.yaml`)
	configFileYaml, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatalln(`Config file`, configFilePath, `doesn't exist.`)
	}

	yamlConfig := YamlConfig{}
	err = yaml.Unmarshal(configFileYaml, &yamlConfig)
	if err != nil {
		log.Fatalln(`Failed to parse config file. Error: `, err)
	}

	config := Config{}

	rolesDirPath := fmt.Sprintf(`%s%s`, configDir, `roles/`)
	rolesDir, err := os.ReadDir(rolesDirPath)
	if err != nil {
		log.Fatalln(`Roles directory`, rolesDirPath, `doesn't exist.`)
	}

	var roles = make(map[string]bool)
	for _, file := range rolesDir {
		if file.IsDir() {
			roles[file.Name()] = true
		}
	}

	config.RolesPath = fmt.Sprintf(`%s/`, strings.TrimRight(rolesDirPath, `/`))

	hostsDirPath := fmt.Sprintf(`%s%s`, configDir, `hosts/`)
	hostsDir, err := os.ReadDir(hostsDirPath)
	if err != nil {
		log.Fatalln(`hosts directory`, hostsDirPath, `doesn't exist.`)
	}

	config.Hosts = loadHostFiles(hostsDir, hostsDirPath, roles)

	config.Listen.Host, _ = cmd.Flags().GetString(`host`)
	if !cmd.Flags().Changed(`host`) && yamlConfig.Listen.Host != `` {
		config.Listen.Host = yamlConfig.Listen.Host
	}

	config.Listen.Port, _ = cmd.Flags().GetUint16(`port`)
	if !cmd.Flags().Changed(`port`) && yamlConfig.Listen.Port != 0 {
		config.Listen.Port = yamlConfig.Listen.Port
	}

	config.Listen.SSLCert, _ = cmd.Flags().GetString(`ssl-cert`)
	if !cmd.Flags().Changed(`ssl-cert`) && yamlConfig.Listen.SSLCert != `` {
		config.Listen.SSLCert = yamlConfig.Listen.SSLCert
	}

	config.Listen.SSLKey, _ = cmd.Flags().GetString(`ssl-key`)
	if !cmd.Flags().Changed(`ssl-key`) && yamlConfig.Listen.SSLKey != `` {
		config.Listen.SSLKey = yamlConfig.Listen.SSLKey
	}

	config.GeneratedConfigDirPath, _ = cmd.Flags().GetString(`generated-config-dir`)
	if !cmd.Flags().Changed(`generated-config-dir`) && yamlConfig.GeneratedConfigDirPath != `` {
		config.GeneratedConfigDirPath = yamlConfig.GeneratedConfigDirPath
	}

	config.URLPrefix, _ = cmd.Flags().GetString(`url-prefix`)
	if !cmd.Flags().Changed(`url-prefix`) && yamlConfig.URLPrefix != `` {
		config.URLPrefix = yamlConfig.URLPrefix
	}

	config.GlobalVars, _ = yaml.Marshal(yamlConfig.GlobalVars)

	return config
}

func loadHostFiles(hostsDir []os.DirEntry, hostsDirPath string, roles map[string]bool) map[string]HostConfig {
	hosts := make(map[string]HostConfig)

	for _, file := range hostsDir {
		yamlConfig := HostYamlConfig{}

		filePath := fmt.Sprintf(`%s%s`, hostsDirPath, file.Name())
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalln(`Failed to read file`, filePath)
		}

		err = yaml.Unmarshal(fileContent, &yamlConfig)
		if err != nil {
			log.Fatalln(`Failed to parse host file`, file.Name(), ` Error: `, err)
		}

		if yamlConfig.ConfigKey == `` {
			log.Fatalln(`Host file`, file.Name(), `doesn't have the config_key set'`)
		}

		hostConfig := HostConfig{
			Name:          strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
			Key:           yamlConfig.ConfigKey,
			HasProjects:   len(yamlConfig.Projects) > 0,
			VariablesFile: fileContent,
		}

		for key := range yamlConfig.Services {
			if !roles[key] {
				log.Fatalln(`Role for service`, key, `on host`, hostConfig.Name, `doesn't exist.'`)
			}

			hostConfig.ServiceNames = append(hostConfig.ServiceNames, key)
		}

		hosts[hostConfig.Name] = hostConfig
	}

	return hosts
}
