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

// SprintData struct
type SprintData struct {
	Name             string
	CompletedIssues  []IssuePrinted
	IncompleteIssues []IssuePrinted
	IssueTypes       []string
}

// IssuePrinted struct
type IssuePrinted struct {
	JiraIssue jira.Issue
	Printed   string
}

//ReleaseNotes struct
type ReleaseNotes struct {
	AllIssues      []jira.Issue
	FilteredIssues []jira.Issue
}

// ProjectsList holds a comma separated list of boards
var ProjectsList string

// ReleaseKey is the key shared among the sprints
var ReleaseKey string

// ReleaseLabel issues with this label should be included in public release notes
var ReleaseLabel string

// Query holds a custom query string for creating custom release notes.
var Query string

// FilterID holds the ID of the pre-built filter querying release notes
var FilterID int

// releasenotesCmd represents the releasenotes command
var releasenotesCmd = &cobra.Command{
	Use:   "releasenotes",
	Short: "Generates release notes for a project and set of releases",
	Long: `By naming Jira releases <projectkey> <sprintkey>, this program
	can generate release notes for all projects listed and the releases.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if Query == "" && FilterID == 0 {
			if ProjectsList == "" {
				log.Fatal("You must specify a project or list of projects with the -p or --projects string flag")
			}

			if ReleaseKey == "" {
				log.Fatal("You must specify a common release key -k or --releasekey string flag")
			}
		} else {
			if ProjectsList != "" {
				fmt.Println("Ignoring -p/--projects due to -q/--query parameter")
			}

			if ReleaseKey != "" {
				fmt.Println("Ignoring -k/--releasekey due to -q/--query parameter")
			}
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
		generateReleaseNotes(jiraClient)
	},
}

func init() {
	rootCmd.AddCommand(releasenotesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// releasenotesCmd.PersistentFlags().String("foo", "", "A help for foo")
	releasenotesCmd.PersistentFlags().StringVarP(&ProjectsList, "projects", "p", "", "comma-separated list of Jira Projects to evaluate")
	releasenotesCmd.PersistentFlags().StringVarP(&ReleaseKey, "releasekey", "k", "", "shared key among all sprints for release names")
	releasenotesCmd.PersistentFlags().StringVarP(&ReleaseLabel, "releaselabel", "l", "", "issues with this label should be included in public release notes")
	releasenotesCmd.PersistentFlags().StringVarP(&Query, "query", "q", "", "custom query (forces ignore of -p and -k)")
	releasenotesCmd.PersistentFlags().IntVarP(&FilterID, "filterid", "f", 0, "Use a custom filter to fetch release notes results")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// releasenotesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getPrintedIssue(i *jira.Issue, baseURL string) string {
	assignee := "UNASSIGNED"
	if i.Fields.Assignee != nil {
		assignee = i.Fields.Assignee.DisplayName
	}
	fmt.Println(i.Fields.Parent.Key)
	markdownIssue := fmt.Sprintf("- [%s](%s/browse/%s)(%s) %s -- %s -- %s\n", i.Key, baseURL, i.Key, i.Fields.Type.Name, i.Fields.Summary, assignee, i.Fields.Status.Name)
	return markdownIssue
}

func generateReleasesString(projectsList string, releaseKey string) string {

	var sb strings.Builder
	projects := strings.Split(projectsList, ",")

	for _, project := range projects {
		sb.WriteString("\"")
		sb.WriteString(project)
		sb.WriteString(" ")
		sb.WriteString(releaseKey)
		sb.WriteString("\", ")
	}
	trimmed := strings.TrimSuffix(sb.String(), ", ")
	return trimmed
}

func getAllAndFilteredReleaseNotes(jiraClient *jira.Client, allQueryString string, filteredQueryString string) ReleaseNotes {
	// set total results > max results so it gets the request the first time
	resultsPerPage := 100

	allSearchOpts := jira.SearchOptions{
		MaxResults: resultsPerPage,
		StartAt:    0,
	}
	filteredSearchOpts := jira.SearchOptions{
		MaxResults: resultsPerPage,
		StartAt:    0,
	}
	var releaseNotes ReleaseNotes
	var allIssues []jira.Issue
	var filteredIssues []jira.Issue

	for {
		allIssuesPage, resp, err := jiraClient.Issue.Search(allQueryString, &allSearchOpts)
		if err != nil {
			log.Fatal(err)
		}
		allIssues = append(allIssues, allIssuesPage...)

		if resp.Total < (resp.StartAt + resp.MaxResults) {
			break
		}
		allSearchOpts.StartAt = allSearchOpts.StartAt + resultsPerPage

	}
	releaseNotes.AllIssues = allIssues

	if filteredQueryString != "" {
		for {
			filteredIssuesPage, resp, err := jiraClient.Issue.Search(filteredQueryString, &filteredSearchOpts)
			if err != nil {
				log.Fatal(err)
			}
			filteredIssues = append(filteredIssues, filteredIssuesPage...)

			if resp.Total < (resp.StartAt + resp.MaxResults) {
				break
			}
			filteredSearchOpts.StartAt = filteredSearchOpts.StartAt + resultsPerPage

		}
		releaseNotes.FilteredIssues = filteredIssues
	}

	return releaseNotes
}

func getFilterReleaseNotes(jiraClient *jira.Client, filterID int) ReleaseNotes {

	jiraFilter, _, err := jiraClient.Filter.Get(filterID)
	if err != nil {
		log.Fatal(err)
	}
	jql := jiraFilter.Jql

	return getCustomReleaseNotes(jiraClient, jql)

}

func getCustomReleaseNotes(jiraClient *jira.Client, queryString string) ReleaseNotes {
	filteredIssuesSearchJQL := ""
	if ReleaseLabel != "" {
		filteredIssuesSearchJQL = "labels = " + ReleaseLabel + " AND " + queryString
	}

	return getAllAndFilteredReleaseNotes(jiraClient, queryString, filteredIssuesSearchJQL)
}

func getIssuesForReleases(jiraClient *jira.Client, releasesString string) ReleaseNotes {

	allIssuesSearchJQL := "fixVersion in (" + releasesString + ") AND status = Done ORDER BY issuetype ASC"
	filteredIssuesSearchJQL := ""
	if ReleaseLabel != "" {
		filteredIssuesSearchJQL = "fixVersion in (" + releasesString + ") AND status = Done AND labels = " + ReleaseLabel + " ORDER BY issuetype ASC"
	}

	return getAllAndFilteredReleaseNotes(jiraClient, allIssuesSearchJQL, filteredIssuesSearchJQL)
}

func generateReleaseNotes(jiraClient *jira.Client) {
	baseURL := fmt.Sprintf("https://%s", jiraClient.GetBaseURL().Host)
	releasesString := generateReleasesString(ProjectsList, ReleaseKey)
	var releaseNotes ReleaseNotes
	var sb strings.Builder
	if Query != "" {
		releaseNotes = getCustomReleaseNotes(jiraClient, Query)
	} else if FilterID > 0 {
		releaseNotes = getFilterReleaseNotes(jiraClient, FilterID)
	} else {
		releaseNotes = getIssuesForReleases(jiraClient, releasesString)
	}

	if ReleaseLabel != "" {
		sb.WriteString("# Public Release Notes (" + ReleaseLabel + ")\n\n")

		for _, pIssue := range releaseNotes.FilteredIssues {
			sb.WriteString(getPrintedIssue(&pIssue, baseURL))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("# All Release Notes\n\n")
	for _, aIssue := range releaseNotes.AllIssues {
		sb.WriteString(getPrintedIssue(&aIssue, baseURL))
	}

	fmt.Println(sb.String())
}
