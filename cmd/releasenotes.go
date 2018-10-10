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
	CompletedIssues  []jira.Issue
	IncompleteIssues []jira.Issue
}

// ProjectsList holds a comma separated list of boards
var ProjectsList string

// ActiveSprint forces the program to create release notes for the currently active sprints
var ActiveSprint bool

// SeparateProjects shows the data in separate projects vs all together
var SeparateProjects bool

// Confluence display the data in Confluence vs Confluence Wiki
var Confluence bool

// SprintsBack is the number of sprints to look back at the data
var SprintsBack int

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
		if SprintsBack < 0 {
			log.Fatal("sprintsback cannot be less than 0")
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
	releasenotesCmd.PersistentFlags().BoolVarP(&ActiveSprint, "active", "a", false, "create release notes for the active sprint")
	releasenotesCmd.PersistentFlags().IntVarP(&SprintsBack, "sprintsback", "b", 0, "number of sprints to look back (defaults to 0, most recent completed sprint)")
	releasenotesCmd.PersistentFlags().BoolVarP(&SeparateProjects, "separate", "s", false, "separate the projects out into individual release notes")
	releasenotesCmd.PersistentFlags().BoolVarP(&Confluence, "confluence", "c", false, "output in confluence wiki format, defaults to markdown")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// releasenotesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func printIssue(i *jira.Issue, baseURL string) {
	assignee := "UNASSIGNED"
	if i.Fields.Assignee != nil {
		assignee = i.Fields.Assignee.DisplayName
	}

	if Confluence {
		fmt.Printf("|[%s|%s/browse/%s]|%s|%s|%s|\n", i.Key, baseURL, i.Key, i.Fields.Summary, assignee, i.Fields.Status.Name)
	} else {
		fmt.Printf("- [%s](%s/browse/%s) %s -- %s -- %s\n", i.Key, baseURL, i.Key, i.Fields.Summary, assignee, i.Fields.Status.Name)
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

	projects := strings.Split(ProjectsList, ",")
	for _, project := range projects {
		thisSprintData := getSprintDataForBoardWithSprintOptions(jiraClient, project, sprintOpts)
		allSprints = append(allSprints, thisSprintData)
		combinedSprints.CompletedIssues = append(combinedSprints.CompletedIssues, thisSprintData.CompletedIssues...)
		combinedSprints.IncompleteIssues = append(combinedSprints.IncompleteIssues, thisSprintData.IncompleteIssues...)
	}

	if SeparateProjects {
		for _, sprint := range allSprints {
			if Confluence {
				fmt.Printf("h1. %s\n\n", sprint.Name)
				fmt.Printf("h2. Done\n\n")
				fmt.Printf("||Key||Summary||Assignee||Status||\n")
			} else {
				fmt.Printf("# %s\n\n", sprint.Name)
				fmt.Printf("## Done\n\n")
			}

			for _, i := range sprint.CompletedIssues {
				printIssue(&i, baseURL)
			}
			if Confluence {
				fmt.Printf("\nh2. Incomplete\n\n")
				fmt.Printf("||Key||Summary||Assignee||Status||\n")
			} else {
				fmt.Printf("\n## Incomplete\n\n")
			}

			for _, i := range sprint.IncompleteIssues {
				printIssue(&i, baseURL)
			}
			fmt.Printf("\n\n")
		}
	} else {
		var sprintNames []string
		for _, sp := range allSprints {
			sprintNames = append(sprintNames, sp.Name)
		}
		sprintNameString := strings.Join(sprintNames, ", ")
		if Confluence {
			fmt.Printf("h1. %s\n\n", sprintNameString)
			fmt.Printf("h2. Done\n\n")
			fmt.Printf("||Key||Summary||Assignee||Status||\n")
		} else {
			fmt.Printf("# %s\n\n", sprintNameString)
			fmt.Printf("## Done\n\n")
		}
		for _, i := range combinedSprints.CompletedIssues {
			printIssue(&i, baseURL)
		}

		if Confluence {
			fmt.Printf("\nh2. Incomplete\n\n")
			fmt.Printf("||Key||Summary||Assignee||Status||\n")
		} else {
			fmt.Printf("\n## Incomplete\n\n")
		}
		for _, i := range combinedSprints.IncompleteIssues {
			printIssue(&i, baseURL)
		}
		fmt.Printf("\n\n")
	}

}

func getSprintDataForBoardWithSprintOptions(jiraClient *jira.Client, project string, sprintOptions jira.GetAllSprintsOptions) SprintData {
	var sprintData SprintData

	jiraBoardOpts := jira.BoardListOptions{
		ProjectKeyOrID: project,
	}
	foundBoards, _, err := jiraClient.Board.GetAllBoards(&jiraBoardOpts)
	if err != nil {
		log.Fatal(err)
	}

	sprints, _, sprintsErr := jiraClient.Board.GetAllSprintsWithOptions(foundBoards.Values[0].ID, &sprintOptions)
	if sprintsErr != nil {
		log.Fatal(sprintsErr)
	}

	if SprintsBack > len(sprints.Values) {
		errMsg := fmt.Sprintf("Cannot specify sprintsback = %d, only fetched %d sprints for project %s", SprintsBack, len(sprints.Values), project)
		log.Fatal(errMsg)
	}

	lastSprint := sprints.Values[len(sprints.Values)-(1+SprintsBack)]

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
