// Copyright © 2018 Patrick McDonagh <patrickjmcd@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira"
	jirasetup "github.com/patrickjmcd/jira-tools/jirasetup"
	"github.com/spf13/cobra"
)

//ServicedeskProject is the Jira project name code for the servicedesk project
var ServicedeskProject string

//DaysOfServicedeskItems holds the number of days of history to fetch
var DaysOfServicedeskItems int

//OutputFilePath is the file path to output the data
var OutputFilePath string

// servicedeskCmd represents the servicedesk command
var servicedeskCmd = &cobra.Command{
	Use:   "servicedesk",
	Short: "Retrieves servicedesk Jira issues in the past X number of days",
	Long: `This command will look at each issue in the provided project 
and check which issues have blocking issues that are completed.
	
This is especially useful for Jira Service Desk projects that
are used to create linked issues in other boards`,
	Run: func(cmd *cobra.Command, args []string) {
		if Project == "" {
			log.Fatal("You must include the -p or --project string parameter")
		}

		url, username, apiKey := jirasetup.GetEnvVariablesOrAsk()
		transport := jira.BasicAuthTransport{
			Username: username,
			Password: apiKey,
		}
		jiraClient, err := jira.NewClient(transport.Client(), url)
		if err != nil {
			log.Fatal("Couldn't log on to the Jira server.")
		}

		serviceDeskIssues := getServicedeskIssuesForProject(jiraClient, Project, DaysOfServicedeskItems)
		csvString := generateCSVFromIssueSlice(jiraClient, serviceDeskIssues)
		if OutputFilePath != "" {
			writeCSVToFile(csvString, OutputFilePath)
		} else {

			fmt.Println(csvString)
		}
	},
}

func init() {
	rootCmd.AddCommand(servicedeskCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// servicedeskCmd.PersistentFlags().String("foo", "", "A help for foo")
	servicedeskCmd.PersistentFlags().StringVarP(&Project, "project", "p", "", "Jira project to use")
	servicedeskCmd.PersistentFlags().StringVarP(&OutputFilePath, "output", "o", "", "CSV File to output")
	servicedeskCmd.MarkFlagRequired("project")
	servicedeskCmd.PersistentFlags().IntVarP(&DaysOfServicedeskItems, "days", "d", 7, "Days of history to retreive")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// servicedeskCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getServicedeskIssuesForProject(jiraClient *jira.Client, projectName string, daysOfHistory int) []jira.Issue {
	searchOpts := jira.SearchOptions{
		MaxResults: 999,
	}

	dateAndQuery := fmt.Sprintf(" and createdDate > startOfDay(-%dd)", daysOfHistory)
	if daysOfHistory <= 0 {
		dateAndQuery = ""
	}

	searchQuery := fmt.Sprintf("project=%s%s ORDER BY createdDate DESC", projectName, dateAndQuery)

	servicedeskIssues, _, pErr := jiraClient.Issue.Search(searchQuery, &searchOpts)

	if pErr != nil {
		log.Fatal(pErr)
	}

	return servicedeskIssues
}

func generateCSVFromIssueSlice(jiraClient *jira.Client, issues []jira.Issue) string {
	var csvStringBuilder strings.Builder

	header := "Type,Key,Summary,Status,Assignee,Reporter,Created,Link\n"
	csvStringBuilder.WriteString(header)

	for _, issue := range issues {

		issueLink := fmt.Sprintf("https://%s/browse/%s", jiraClient.GetBaseURL().Host, issue.Key)

		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}

		created := time.Time(issue.Fields.Created)

		issueString := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			issue.Fields.Type.Name,
			issue.Key,
			issue.Fields.Summary,
			issue.Fields.Status.Name,
			assignee,
			issue.Fields.Reporter.DisplayName,
			created.String(),
			issueLink)
		csvStringBuilder.WriteString(issueString)
	}

	return csvStringBuilder.String()
}

func writeCSVToFile(csvString string, file string) {
	createdFile, err := os.Create(file)
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer createdFile.Close()

	fmt.Fprintf(createdFile, csvString)

}
