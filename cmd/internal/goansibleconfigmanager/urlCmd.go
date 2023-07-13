package goansibleconfigmanager

import (
	"fmt"
	"github.com/spf13/cobra"
)

func getURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     `url [flags] [host]...`,
		Aliases: []string{`urls`},
		Short:   `Generate URLs for initializing hosts or downloading configs`,
		Run:     runURLCmd,
	}

	return cmd
}

func runURLCmd(cmd *cobra.Command, args []string) {
	config := loadConfig(cmd)

	hosts := config.GetExistingHostsOrFatal(args)

	firstPrint := true
	for _, hostConfig := range config.Hosts {
		if len(hosts) > 0 && !hosts[hostConfig.Name] {
			continue
		}

		// separate outputs
		if !firstPrint {
			fmt.Println() //nolint:forbidigo
		}

		firstPrint = false

		fmt.Println(hostConfig.Name) //nolint:forbidigo

		fmt.Println("config:", generateURL(config, hostConfig.Name, `config.tar`)) //nolint:forbidigo

		for _, scriptConfig := range config.Scripts {
			scriptURL := generateURL(config, hostConfig.Name, scriptConfig.Name)
			fmt.Printf("script %s: curl -o- %s | bash\n", scriptConfig.Name, scriptURL) //nolint:forbidigo
		}
	}
}

func generateURL(config Config, host string, file string) string {
	hostConfig := config.Hosts[host]

	return fmt.Sprintf(`%s/%s/%s/%s`, config.URLPrefix, hostConfig.Name, hostConfig.Key, file)
}
