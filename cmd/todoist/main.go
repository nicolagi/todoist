package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/nicolagi/todoist"
	log "github.com/sirupsen/logrus"
)

var client *todoist.Client

func main() {
	home := mustHomeDir()
	tokenFile := path.Join(home, "lib/todoist/token")
	wireLogFile := path.Join(home, "lib/todoist/wire.log")
	apiToken := mustReadTokenFile(tokenFile)
	client = mustCreateClient(apiToken, wireLogFile)

	// Create initial window listing all projects.
	newAllProjectsWindow()

	// The program will be terminated when the last acme window owned by this process is deleted.
	select {}
}

func mustHomeDir() string {
	u, err := user.Current()
	if err != nil {
		log.WithField("cause", err).Fatal("Could not get current user")
	}
	return u.HomeDir
}

func mustReadTokenFile(tokenFile string) string {
	logEntry := log.WithField("path", tokenFile)
	fi, err := os.Stat(tokenFile)
	if err != nil {
		logEntry.WithField("cause", err).Fatal("Could not check permissions")
	}
	if fi.Mode()&0077 != 0 {
		logEntry.WithFields(log.Fields{
			"got":  fmt.Sprintf("%#o", fi.Mode()),
			"want": fmt.Sprintf("%#o", fi.Mode()&0700),
		}).Fatal("Stricter permissions required")
	}
	b, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		logEntry.WithField("cause", err).Fatal("Todoist API token not found")
	}
	return strings.TrimSpace(string(b))
}

func mustCreateClient(apiToken string, wireLogFile string) *todoist.Client {
	client, err := todoist.NewClient(apiToken, todoist.WithWireLog(wireLogFile))
	if err != nil {
		log.WithField("cause", err).Fatal("Could not create client")
	}
	if err := client.Load(); err != nil {
		log.WithField("cause", err).Warning("Could not load local data, will do a full sync")
	}
	return client
}
