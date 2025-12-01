package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"git-repository-visualizer/internal/git"
)

func main() {
	// Get repository path from command line argument
	if len(os.Args) != 2 {
		log.Fatal("Please provide the repository path as an argument")
	}
	repoPath := os.Args[1]

	// Analyze repository
	contributors, err := git.GetContributors(repoPath)
	if err != nil {
		log.Fatal(err)
	}

	// Print results
	fmt.Printf("\nRepository: %s\n\n", repoPath)
	fmt.Printf("%-40s %-10s\n", "Author", "Commits")
	fmt.Println(strings.Repeat("-", 50))

	for _, c := range contributors {
		fmt.Printf("%-40s %-10d\n",
			fmt.Sprintf("%s <%s>", c.Name, c.Email),
			c.Commits,
		)
	}
}
