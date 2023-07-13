package goansibleconfigmanager

import (
	"os"
)

func Execute() {
	rootCmd := getServeCmd()
	rootCmd.AddCommand(
		getGenerateCmd(),
		getURLCmd(),
	)

	rootCmd.PersistentFlags().String(`config-dir`, `/etc/ansible-config-manager/`, `config dir path`)

	rootCmd.PersistentFlags().String(`host`, `0.0.0.0`, `host to listen on, can be an interface name`)
	rootCmd.PersistentFlags().Uint16(`port`, 80, `port to listen on`)
	rootCmd.PersistentFlags().String(`ssl-cert`, ``, `SSL certificate path`)
	rootCmd.PersistentFlags().String(`ssl-key`, ``, `SSL certificate key path`)

	rootCmd.PersistentFlags().String(`generated-config-dir`, `/var/lib/ansible-config-manager/`, `dir where generated configs are stored`)

	rootCmd.PersistentFlags().String(`url-prefix`, ``, `prefix to use for generating URLs`)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
