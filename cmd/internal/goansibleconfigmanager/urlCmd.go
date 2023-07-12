package goansibleconfigmanager

import (
	"github.com/spf13/cobra"
	"log"
)

func getURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url",
		Short: "Short generate desc",
		Long:  `Long generate description`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Println(`URL`)
		},
	}

	cmd.Flags().String(`url-prefix`, ``, `prefix to use for generating URLs`)

	return cmd
}
