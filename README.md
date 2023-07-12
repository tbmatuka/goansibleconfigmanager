GoAnsibleConfigManager
===============

GoAnsibleConfigManager is a service for generating and serving Ansible configurations written in Go. It lets you use Ansible in a pull configuration instead of push.

# Features

* Generate a tar file with a config for each host
* Serve config file on a HTTP API
* Serve an initial setup script for each host, so that you don't need to copy/paste one on each new server
* Generate URLs for scripts and tar packages

# Installation

At the moment your only options are building the app yourself or downloading the [release](https://github.com/tbmatuka/goansibleconfigmanager/releases/latest) package which contains a built binary.
