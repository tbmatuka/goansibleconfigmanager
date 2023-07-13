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
		SSL     bool
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

	// Names of scripts as keys and paths to scripts as calues
	Scripts map[string]ScriptConfig
}

type YamlConfig struct {
	Listen struct {
		Host    string `yaml:"host"`
		Port    uint16 `yaml:"port"`
		SSLCert string `yaml:"ssl_cert"`
		SSLKey  string `yaml:"ssl_key"`
	} `yaml:"listen"`

	URLPrefix string `yaml:"url_prefix"`

	GeneratedConfigDirPath string `yaml:"generated_config_dir_path"`

	GlobalVars map[string]interface{} `yaml:"global_vars"`
}

type ScriptConfig struct {
	Name     string
	Path     string
	Template bool
}

type HostConfig struct {
	Name          string
	Key           string
	ServiceNames  []string
	HasProjects   bool
	VariablesFile []byte
	TarFilePath   string
}

type HostYamlConfig struct {
	ConfigKey string                 `yaml:"config_key"`
	Services  map[string]interface{} `yaml:"services"`
	Projects  map[string]interface{} `yaml:"projects"`
}

func (config *Config) GetExistingHostsOrFatal(hosts []string) map[string]bool {
	var hostsMap = make(map[string]bool)

	for _, host := range hosts {
		if _, exists := config.Hosts[host]; !exists {
			log.Fatalln(`Host`, host, `does not exist.`)
		}

		hostsMap[host] = true
	}

	return hostsMap
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

	config.CheckSSL()
	config.DetermineURLPrefix(cmd, yamlConfig)

	config.GeneratedConfigDirPath, _ = cmd.Flags().GetString(`generated-config-dir`)
	if !cmd.Flags().Changed(`generated-config-dir`) && yamlConfig.GeneratedConfigDirPath != `` {
		config.GeneratedConfigDirPath = yamlConfig.GeneratedConfigDirPath
	}

	config.LoadHostFiles(fmt.Sprintf(`%s%s`, configDir, `hosts/`), roles)
	config.LoadScripts(fmt.Sprintf(`%s%s`, configDir, `scripts/`))

	config.GlobalVars, _ = yaml.Marshal(yamlConfig.GlobalVars)

	return config
}

func (config *Config) LoadHostFiles(hostsDirPath string, roles map[string]bool) {
	hostsDir, err := os.ReadDir(hostsDirPath)
	if err != nil {
		log.Fatalln(`hosts directory`, hostsDirPath, `doesn't exist.`)
	}

	tarFileDir := strings.TrimRight(config.GeneratedConfigDirPath, `/`)

	config.Hosts = make(map[string]HostConfig)

	for _, file := range hostsDir {
		fileExtension := filepath.Ext(file.Name())
		if fileExtension != `.yml` && fileExtension != `.yaml` {
			continue
		}

		hostName := strings.TrimSuffix(file.Name(), fileExtension)

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
			Name:          hostName,
			Key:           yamlConfig.ConfigKey,
			HasProjects:   len(yamlConfig.Projects) > 0,
			VariablesFile: fileContent,
			TarFilePath:   fmt.Sprintf(`%s/%s.tar`, tarFileDir, hostName),
		}

		for key := range yamlConfig.Services {
			if !roles[key] {
				log.Fatalln(`Role for service`, key, `on host`, hostConfig.Name, `doesn't exist.'`)
			}

			hostConfig.ServiceNames = append(hostConfig.ServiceNames, key)
		}

		config.Hosts[hostConfig.Name] = hostConfig
	}
}

func (config *Config) LoadScripts(scriptsDirPath string) {
	config.Scripts = make(map[string]ScriptConfig)

	dir, err := os.ReadDir(scriptsDirPath)
	if err != nil {
		return
	}

	for _, file := range dir {
		scriptConfig := ScriptConfig{
			Name: file.Name(),
			Path: fmt.Sprintf(`%s%s`, scriptsDirPath, file.Name()),
		}

		fileExtension := filepath.Ext(file.Name())
		if fileExtension == `.tpl` {
			scriptConfig.Name = strings.TrimSuffix(file.Name(), fileExtension)
			scriptConfig.Template = true
		}

		config.Scripts[scriptConfig.Name] = scriptConfig
	}
}

func (config *Config) CheckSSL() {
	if config.Listen.SSLCert != `` && config.Listen.SSLKey == `` {
		log.Fatalln(`SSL certificate is defined, but the SSL key is not.`)
	}

	if config.Listen.SSLCert == `` && config.Listen.SSLKey != `` {
		log.Fatalln(`SSL key is defined, but the SSL certificate is not.`)
	}

	config.Listen.SSL = config.Listen.SSLCert != ``
}

func (config *Config) DetermineURLPrefix(cmd *cobra.Command, yamlConfig YamlConfig) {
	urlPrefix, _ := cmd.Flags().GetString(`url-prefix`)
	if !cmd.Flags().Changed(`url-prefix`) && yamlConfig.URLPrefix != `` {
		urlPrefix = yamlConfig.URLPrefix
	}

	urlPrefix = strings.TrimRight(urlPrefix, `/`)

	if urlPrefix == `` {
		urlSchema := `http`
		if config.Listen.SSL {
			urlSchema = `https`
		}

		urlPort := fmt.Sprintf(`:%d`, config.Listen.Port)
		if (config.Listen.Port == 80 && !config.Listen.SSL) || (config.Listen.Port == 443 && config.Listen.SSL) {
			urlPort = ``
		}

		urlPrefix = fmt.Sprintf(`%s://%s%s`, urlSchema, getListenAddress(*config), urlPort)
	}

	config.URLPrefix = urlPrefix
}
