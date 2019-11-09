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

	// viper.SetEnvPrefix("jira")
	// viper.BindEnv("username")
	// viper.BindEnv("url")
	// viper.BindEnv("api_key")

	jiraURL = viper.GetString("jira_url")
	if !viper.IsSet("jira_url") {
		jiraURL = getUnspecifiedKey("Jira URL")
		// os.Setenv("JIRA_URL", jiraURL)
		viper.Set("jira_url", jiraURL)
	}

	jiraUsername = viper.GetString("jira_username")
	if !viper.IsSet("jira_username") {
		jiraUsername = getUnspecifiedKey("Jira Username")
		// os.Setenv("JIRA_USERNAME", jiraUsername)
		viper.Set("jira_username", jiraUsername)
	}

	jiraAPIKey = viper.GetString("jira_api_key")
	if !viper.IsSet("jira_api_key") {
		jiraAPIKey = getUnspecifiedKey("Jira API Key")
		// os.Setenv("JIRA_API_KEY", jiraAPIKey)
		viper.Set("jira_api_key", jiraAPIKey)
		fmt.Println()
	}
	viper.WriteConfig()

	return jiraURL, jiraUsername, jiraAPIKey
}
