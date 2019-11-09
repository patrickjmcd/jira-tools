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
	"strings"

	jira "github.com/andygrunwald/go-jira"
	jirasetup "github.com/patrickjmcd/jira-tools/jirasetup"
	"github.com/spf13/cobra"
)

// AssignedProjectsList is the comma-separated list of projects to include
var AssignedProjectsList string

// AssignedExcludeProjectsList is the comma-separated list of projects to exclude
var AssignedExcludeProjectsList string

// releasenotesCmd represents the releasenotes command
var assignedCmd = &cobra.Command{
	Use:   "mine",
	Short: "Generates a list of issues assigned to the current user",
	Long: `The list can be filtered by explicitly specifying projects or excluding projects
	`,
	Run: func(cmd *cobra.Command, args []string) {
		url, username, apiKey := jirasetup.GetEnvVariablesOrAsk()
		transport := jira.BasicAuthTransport{
			Username: username,
			Password: apiKey,
		}
		jiraClient, err := jira.NewClient(transport.Client(), url)
		if err != nil {
			log.Fatal("Couldn't log on to the Jira server.")
		}
		allIssues := getAssignedIssues(jiraClient)

		for _, issue := range allIssues {
			printIssue(&issue, url)
		}

	},
}

func init() {
	rootCmd.AddCommand(assignedCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// releasenotesCmd.PersistentFlags().String("foo", "", "A help for foo")
	assignedCmd.PersistentFlags().StringVarP(&AssignedProjectsList, "include-projects", "i", "", "comma-separated list of Jira Projects to include")
	assignedCmd.PersistentFlags().StringVarP(&AssignedExcludeProjectsList, "exclude-projects", "x", "", "comma-separated list of Jira Projects to exclude")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// releasenotesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getAssignedIssues(jiraClient *jira.Client) []jira.Issue {
	var allIssues []jira.Issue
	queryString := makeQueryString()

	// set total results > max results so it gets the request the first time
	resultsPerPage := 100

	searchOpts := jira.SearchOptions{
		MaxResults: resultsPerPage,
		StartAt:    0,
	}

	for {
		allIssuesPage, resp, err := jiraClient.Issue.Search(queryString, &searchOpts)
		if err != nil {
			log.Fatal(err)
		}
		allIssues = append(allIssues, allIssuesPage...)

		if resp.Total < (resp.StartAt + resp.MaxResults) {
			break
		}
		searchOpts.StartAt = searchOpts.StartAt + resultsPerPage

	}
	return allIssues

}

func makeQueryString() string {
	var sb strings.Builder
	sb.WriteString("assignee = currentUser() AND resolution IS EMPTY")
	if AssignedProjectsList != "" {
		sb.WriteString(" AND project in (")
		sb.WriteString(AssignedProjectsList)
		sb.WriteString(")")
	} else if AssignedExcludeProjectsList != "" {
		sb.WriteString(" AND project NOT in (")
		sb.WriteString(AssignedExcludeProjectsList)
		sb.WriteString(")")
	}
	return sb.String()
}

func printIssue(i *jira.Issue, baseURL string) {
	markdownIssue := fmt.Sprintf("- [%s] %s (%s) -- %s\n\t(%s/browse/%s)", i.Key, i.Fields.Summary, i.Fields.Type.Name, i.Fields.Status.Name, baseURL, i.Key)
	fmt.Println(markdownIssue)
}
