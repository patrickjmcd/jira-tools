// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	CompletedIssues  []jira.Issue
	IncompleteIssues []jira.Issue
}

// Boardslist holds a comma separated list of boards
var Boardslist string

// ActiveSprint forces the program to create release notes for the currently active sprints
var ActiveSprint bool

// SeparateProjects shows the data in separate projects vs all together
var SeparateProjects bool

// Markdown display the data in markdown vs Confluence Wiki
var Markdown bool

// releasenotesCmd represents the releasenotes command
var releasenotesCmd = &cobra.Command{
	Use:   "releasenotes",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		url, username, password := jirasetup.GetEnvVariablesOrAsk()
		transport := jira.BasicAuthTransport{
			Username: username,
			Password: password,
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
	releasenotesCmd.PersistentFlags().StringVarP(&Boardslist, "boards", "b", "", "list of Jira boards to evaluate")
	releasenotesCmd.PersistentFlags().BoolVarP(&ActiveSprint, "active", "a", false, "create release notes for the active sprint")
	releasenotesCmd.PersistentFlags().BoolVarP(&SeparateProjects, "separate", "s", false, "separate the projects out into individual release notes")
	releasenotesCmd.PersistentFlags().BoolVarP(&Markdown, "markdown", "m", false, "output in markdown, defaults to confluence wiki")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// releasenotesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func printIssue(i *jira.Issue, baseURL string) {
	assignee := "UNASSIGNED"
	if i.Fields.Assignee != nil {
		assignee = i.Fields.Assignee.DisplayName
	}

	if Markdown {
		fmt.Printf("  * [%s|%s/browse/%s] %s -- %s -- %s\n", i.Key, baseURL, i.Key, i.Fields.Summary, assignee, i.Fields.Status.Name)
	} else {
		fmt.Printf("|[%s|%s/browse/%s]|%s|%s|%s|\n", i.Key, baseURL, i.Key, i.Fields.Summary, assignee, i.Fields.Status.Name)
	}

}

func generateReleaseNotes(jiraClient *jira.Client) {
	var combinedSprints SprintData
	var allSprints []SprintData
	baseURL := fmt.Sprintf("https://%s", jiraClient.GetBaseURL().Host)
	sprintOpts := jira.GetAllSprintsOptions{
		State: "closed",
	}

	if ActiveSprint {
		sprintOpts.State = "active"
	}

	combinedSprints.Name = fmt.Sprintf("Combined Data for %s Projects", Boardslist)
	boards := strings.Split(Boardslist, ",")
	for _, board := range boards {
		thisSprintData := getSprintDataForBoardWithSprintOptions(jiraClient, board, sprintOpts)
		allSprints = append(allSprints, thisSprintData)
		combinedSprints.CompletedIssues = append(combinedSprints.CompletedIssues, thisSprintData.CompletedIssues...)
		combinedSprints.IncompleteIssues = append(combinedSprints.IncompleteIssues, thisSprintData.IncompleteIssues...)
	}

	if SeparateProjects {
		for _, sprint := range allSprints {
			if Markdown {
				fmt.Printf("# %s\n\n", sprint.Name)
				fmt.Printf("## Done\n\n")
			} else {
				fmt.Printf("h1. %s\n\n", sprint.Name)
				fmt.Printf("h2. Done\n\n")
				fmt.Printf("||Key||Summary||Assignee||Status||\n")
			}

			for _, i := range sprint.CompletedIssues {
				printIssue(&i, baseURL)
			}
			if Markdown {
				fmt.Printf("\n## Incomplete\n\n")
			} else {
				fmt.Printf("\nh2. Incomplete\n\n")
				fmt.Printf("||Key||Summary||Assignee||Status||\n")
			}

			for _, i := range sprint.IncompleteIssues {
				printIssue(&i, baseURL)
			}
			fmt.Printf("\n\n")
		}
	} else {
		if Markdown {
			fmt.Printf("# %s\n\n", combinedSprints.Name)
			fmt.Printf("## Done\n\n")
		} else {
			fmt.Printf("h1. %s\n\n", combinedSprints.Name)
			fmt.Printf("h2. Done\n\n")
			fmt.Printf("||Key||Summary||Assignee||Status||\n")
		}
		for _, i := range combinedSprints.CompletedIssues {
			printIssue(&i, baseURL)
		}

		if Markdown {
			fmt.Printf("\n## Incomplete\n\n")
		} else {
			fmt.Printf("\nh2. Incomplete\n\n")
			fmt.Printf("||Key||Summary||Assignee||Status||\n")
		}
		for _, i := range combinedSprints.IncompleteIssues {
			printIssue(&i, baseURL)
		}
		fmt.Printf("\n\n")
	}

}

func getSprintDataForBoardWithSprintOptions(jiraClient *jira.Client, board string, sprintOptions jira.GetAllSprintsOptions) SprintData {
	var sprintData SprintData

	jiraBoardOpts := jira.BoardListOptions{
		ProjectKeyOrID: board,
	}
	foundBoards, _, err := jiraClient.Board.GetAllBoards(&jiraBoardOpts)
	if err != nil {
		log.Fatal(err)
	}

	sprints, _, sprintsErr := jiraClient.Board.GetAllSprintsWithOptions(foundBoards.Values[0].ID, &sprintOptions)
	if sprintsErr != nil {
		log.Fatal(sprintsErr)
	}

	lastSprint := sprints.Values[len(sprints.Values)-1]

	sprintData.Name = lastSprint.Name
	issues, _, issueErr := jiraClient.Sprint.GetIssuesForSprint(lastSprint.ID)
	if issueErr != nil {
		log.Fatal(issueErr)
	}

	for _, issue := range issues {
		switch issue.Fields.Status.Name {
		case "In Progress", "To Do":
			sprintData.IncompleteIssues = append(sprintData.IncompleteIssues, issue)

		default:
			sprintData.CompletedIssues = append(sprintData.CompletedIssues, issue)
		}
	}

	return sprintData

}
