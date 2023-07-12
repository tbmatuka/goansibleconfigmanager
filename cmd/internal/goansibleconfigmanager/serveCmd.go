package goansibleconfigmanager

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"net"
	"net/http"
	"time"
)

func getServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ansible-config-manager",
		Short: "Short serve desc",
		Long:  `Long serve description`,
		Run: func(cmd *cobra.Command, args []string) {
			config := loadConfig(cmd)

			host := config.Listen.Host

			networkInterface, err := net.InterfaceByName(config.Listen.Host)
			if err == nil {
				log.Println("Found interface by name:", networkInterface.Name)

				addresses, _ := networkInterface.Addrs()
				firstAddress, ok := addresses[0].(*net.IPNet)

				if !ok {
					log.Fatalln("Failed to get address for interface:", networkInterface.Name)
				}

				host = firstAddress.IP.String()
			}

			server := &http.Server{
				Addr:              fmt.Sprintf(`%s:%d`, host, config.Listen.Port),
				ReadHeaderTimeout: 3 * time.Second,
			}

			log.Println("Listening on:", server.Addr)

			if config.Listen.SSLCert != `` && config.Listen.SSLKey != `` {
				log.Fatal(server.ListenAndServeTLS(config.Listen.SSLCert, config.Listen.SSLKey))
			} else { //nolint:revive
				log.Fatal(server.ListenAndServe())
			}
		},
	}

	return cmd
}
