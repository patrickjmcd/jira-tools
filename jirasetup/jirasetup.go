package jirasetup

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
)

func getUnspecifiedKey(key string) string {
	var byteRead []byte
	var stringRead string
	var err error
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", key)
	if key == "Jira API Key" {
		byteRead, err = terminal.ReadPassword(int(syscall.Stdin))
		stringRead = string(byteRead)
	} else {
		stringRead, err = reader.ReadString('\n')
	}

	if err != nil {
		log.Fatalf("You need to specify a %s\n", key)
	}
	trimmedVal := strings.TrimSuffix(stringRead, "\n")
	return trimmedVal
}

// GetEnvVariablesOrAsk -- returns jira env variables
func GetEnvVariablesOrAsk() (string, string, string) {
	var jiraURL string
	var jiraUsername string
	var jiraAPIKey string

	viper.SetEnvPrefix("jira")
	viper.BindEnv("username")
	viper.BindEnv("url")
	viper.BindEnv("api_key")

	jiraURL = viper.GetString("url")
	if !viper.IsSet("url") {
		jiraURL = getUnspecifiedKey("Jira URL")
		os.Setenv("JIRA_URL", jiraURL)
	}

	jiraUsername = viper.GetString("username")
	if !viper.IsSet("username") {
		jiraUsername = getUnspecifiedKey("Jira Username")
		os.Setenv("JIRA_USERNAME", jiraUsername)
	}

	jiraAPIKey = viper.GetString("api_key")
	if !viper.IsSet("api_key") {
		jiraAPIKey = getUnspecifiedKey("Jira API Key")
		os.Setenv("JIRA_API_KEY", jiraAPIKey)
	}

	return jiraURL, jiraUsername, jiraAPIKey
}
