package goansibleconfigmanager

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

type ScriptRenderContext struct {
	Host      HostConfig
	ConfigURL string
}

func getServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   `ansible-config-manager`,
		Short: `Serve host config files over HTTP`,
		Run:   runServeCmd,
	}

	return cmd
}

func runServeCmd(cmd *cobra.Command, _ []string) {
	config := loadConfig(cmd)

	host := getListenAddress(config)

	server := &http.Server{
		Addr:              fmt.Sprintf(`%s:%d`, host, config.Listen.Port),
		ReadHeaderTimeout: 3 * time.Second,
	}

	http.HandleFunc(`/`, func(responseWriter http.ResponseWriter, request *http.Request) {
		handleRequest(responseWriter, request, config)
	})

	log.Println("Listening on:", server.Addr)

	if config.Listen.SSLCert != `` && config.Listen.SSLKey != `` {
		log.Fatal(server.ListenAndServeTLS(config.Listen.SSLCert, config.Listen.SSLKey))
	} else { //nolint:revive
		log.Fatal(server.ListenAndServe())
	}
}

func handleRequest(responseWriter http.ResponseWriter, request *http.Request, config Config) {
	requestArgs := strings.SplitN(strings.Trim(request.RequestURI, `/`), `/`, 3)

	if len(requestArgs) < 3 {
		http.NotFound(responseWriter, request)

		return
	}

	if requestArgs[0] == `` || requestArgs[1] == `` || requestArgs[2] == `` {
		http.NotFound(responseWriter, request)

		return
	}

	requestHost := requestArgs[0]
	requestKey := requestArgs[1]
	requestFile := requestArgs[2]

	hostConfig := config.Hosts[requestHost]
	if hostConfig.Name == `` {
		http.NotFound(responseWriter, request)

		return
	}

	if requestKey != hostConfig.Key {
		http.NotFound(responseWriter, request)

		return
	}

	if requestFile == `config.tar` {
		tarFileContent, err := os.ReadFile(hostConfig.TarFilePath)
		if err != nil {
			http.Error(responseWriter, `Error reading config`, http.StatusInternalServerError)

			return
		}

		_, _ = responseWriter.Write(tarFileContent)

		return
	}

	scriptConfig := config.Scripts[requestFile]
	if scriptConfig.Name == `` {
		http.NotFound(responseWriter, request)

		return
	}

	scriptSource, err := os.ReadFile(scriptConfig.Path)
	if err != nil {
		http.Error(responseWriter, `Error reading script`, http.StatusInternalServerError)

		return
	}

	if !scriptConfig.Template {
		_, _ = responseWriter.Write(scriptSource)

		return
	}

	textTemplate, err := template.New(scriptConfig.Name).Parse(string(scriptSource))
	if err != nil {
		http.Error(responseWriter, `Error parsing script template`, http.StatusInternalServerError)

		return
	}

	var renderBuffer bytes.Buffer
	renderWriter := bufio.NewWriter(&renderBuffer)

	renderContext := ScriptRenderContext{
		Host:      hostConfig,
		ConfigURL: generateURL(config, hostConfig.Name, `config.tar`),
	}

	err = textTemplate.Execute(renderWriter, renderContext)
	if err != nil {
		http.Error(responseWriter, `Error rendering script template`, http.StatusInternalServerError)

		return
	}

	_ = renderWriter.Flush()
	_, _ = responseWriter.Write(renderBuffer.Bytes())
}

func getListenAddress(config Config) string {
	networkInterface, err := net.InterfaceByName(config.Listen.Host)
	if err == nil {
		addresses, _ := networkInterface.Addrs()
		firstAddress, ok := addresses[0].(*net.IPNet)

		if !ok {
			log.Fatalln("Failed to get address for interface:", networkInterface.Name)
		}

		return firstAddress.IP.String()
	}

	return config.Listen.Host
}
