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

	jira "github.com/andygrunwald/go-jira"
	"github.com/fatih/color"
	jirasetup "github.com/patrickjmcd/jira-tools/jirasetup"
	"github.com/spf13/cobra"
)

// Verbose prints out the options
var Verbose bool

//Project is the Jira project name code
var Project string

// unblockedCmd represents the unblocked command
var unblockedCmd = &cobra.Command{
	Use:   "unblocked",
	Short: "Retrieves unblocked Jira issues",
	Long: `This command will look at each issue in the provided project 
and check which issues have blocking issues that are completed.
	
This is especially useful for Jira Service Desk projects that
are used to create linked issues in other boards`,
	Run: func(cmd *cobra.Command, args []string) {
		if Project == "" {
			log.Fatal("You must include the -p or --project string parameter")
		}

		url, username, password := jirasetup.GetEnvVariablesOrAsk()
		transport := jira.BasicAuthTransport{
			Username: username,
			Password: password,
		}
		jiraClient, err := jira.NewClient(transport.Client(), url)
		if err != nil {
			log.Fatal("Couldn't log on to the Jira server.")
		}

		// checkResolvedLinkedIssuesForProject(jiraClient, projectName, verbose)
		issuesWithResolved := getResolvedLinkedIssuesForProject(jiraClient, Project, Verbose)
		if len(issuesWithResolved) > 0 {
			color.Red("------------------------------------------------------")
			color.Red("   The following %d issues have completed linked issues  ", len(issuesWithResolved))
			color.Red("------------------------------------------------------")
			for _, issue := range issuesWithResolved {
				color.Red("[%s] %s - %s/browse/%s", issue.Key, issue.Fields.Summary, url, issue.Key)
			}
			color.Red("------------------------------------------------------")
		} else {
			color.Green("------------------------------------------------------")
			color.Green("  All issues seem to still have pending linked issues. ")
			color.Green("------------------------------------------------------")
		}
	},
}

func init() {
	rootCmd.AddCommand(unblockedCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// unblockedCmd.PersistentFlags().String("foo", "", "A help for foo")
	unblockedCmd.PersistentFlags().StringVarP(&Project, "project", "p", "", "Jira project to use")
	unblockedCmd.MarkFlagRequired("project")
	unblockedCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// unblockedCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func checkLinkedIssueStatus(jiraClient *jira.Client, issue *jira.Issue, c chan string, verbose bool) {
	issueLinks := issue.Fields.IssueLinks
	var linkedIssues []*jira.Issue
	for _, linked := range issueLinks {
		if linked.OutwardIssue != nil {
			linkedIssues = append(linkedIssues, linked.OutwardIssue)
		}
		if linked.InwardIssue != nil {
			linkedIssues = append(linkedIssues, linked.InwardIssue)
		}
	}
	linkedIssuesStillPending := false
	for _, lIssue := range linkedIssues {
		if verbose {
			switch lIssue.Fields.Status.Name {
			case "In Progress":
				color.Set(color.FgBlue)
			case "To Do":
				color.Set(color.FgGreen)
			default:
				color.Set(color.FgRed)
			}

			fmt.Printf(" -- [%s] %s = %+v\n", lIssue.Key, lIssue.Fields.Summary, lIssue.Fields.Status.Name)
			color.Unset()
		}
		if lIssue.Fields.Status.Name == "In Progress" || lIssue.Fields.Status.Name == "To Do" {
			linkedIssuesStillPending = true
		}
	}
	if !linkedIssuesStillPending && len(linkedIssues) > 0 {
		c <- fmt.Sprintf("[%s] %s", issue.Key, issue.Fields.Summary)
	}
}

func getLinkedIssuesForIssue(jiraClient *jira.Client, issue *jira.Issue) []*jira.Issue {
	issueLinks := issue.Fields.IssueLinks
	var linkedIssues []*jira.Issue
	for _, linked := range issueLinks {
		if linked.OutwardIssue != nil {
			linkedIssues = append(linkedIssues, linked.OutwardIssue)
		}
		if linked.InwardIssue != nil {
			linkedIssues = append(linkedIssues, linked.InwardIssue)
		}
	}
	return linkedIssues
}

func checkResolvedLinkedIssuesForProject(jiraClient *jira.Client, projectName string, verbose bool) {
	c := make(chan string)

	searchOpts := jira.SearchOptions{
		MaxResults: 999,
	}

	projectIssues, _, pErr := jiraClient.Issue.Search("project="+projectName+" and resolved is EMPTY", &searchOpts)
	if pErr != nil {
		log.Fatal(pErr)
	}

	for _, issue := range projectIssues {
		go func(issue jira.Issue) {
			checkLinkedIssueStatus(jiraClient, &issue, c, verbose)
		}(issue)
	}

	for l := range c {
		color.Red(l)
	}

}

func getResolvedLinkedIssuesForProject(jiraClient *jira.Client, projectName string, verbose bool) []jira.Issue {
	searchOpts := jira.SearchOptions{
		MaxResults: 999,
	}

	var issuesWithResolvedLinkedIssues []jira.Issue

	projectIssues, _, pErr := jiraClient.Issue.Search("project="+projectName+" and resolved is EMPTY", &searchOpts)

	if pErr != nil {
		log.Fatal(pErr)
	}

	for _, issue := range projectIssues {
		linkedIssues := getLinkedIssuesForIssue(jiraClient, &issue)
		if verbose {
			fmt.Printf("\n[%s] %s -- %d issues\n", issue.Key, issue.Fields.Summary, len(linkedIssues))
		}
		linkedIssuesStillPending := false
		for _, lIssue := range linkedIssues {
			if verbose {
				switch lIssue.Fields.Status.Name {
				case "In Progress":
					color.Set(color.FgBlue)
				case "To Do":
					color.Set(color.FgGreen)
				default:
					color.Set(color.FgRed)
				}
				fmt.Printf(" -- [%s] %s = %+v\n", lIssue.Key, lIssue.Fields.Summary, lIssue.Fields.Status.Name)
				color.Unset()
			}
			if lIssue.Fields.Status.Name == "To Do" || lIssue.Fields.Status.Name == "In Progress" {
				linkedIssuesStillPending = true
			}
		}
		if !linkedIssuesStillPending && len(linkedIssues) > 0 {
			issuesWithResolvedLinkedIssues = append(issuesWithResolvedLinkedIssues, issue)
		}

	}
	if verbose {
		fmt.Println()
		fmt.Println()
		fmt.Println()
	}
	return issuesWithResolvedLinkedIssues
}
