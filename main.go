/*
Copyright © 2024 Austin Sabel austin.sabel@gmail.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"encoding/json"

	"github.com/go-git/go-git/v5"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

// Commit Prompts structure
type CommitPromptData struct {
	CommitType           string `json:"commit_type"`
	Scope                string `json:"scope"`
	ShortDescription     string `json:"short_description"`
	LongDescription      string `json:"long_description"`
	BreakingChange       bool   `json:"breaking_change"`
	BreakingChangeNote   string `json:"breaking_change_note"`
}

// Build Info Vars
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Global Vars
var (
	commitTypes []string
	scopes      []string
	gitRoot     string
)

func gitStatus() {
	repo, err := openGitRepo()
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	Worktree, err := repo.Worktree()
	if err != nil {
		pterm.Fatal.Println("Error opening Git repository:", err)
	}

	gitRoot = Worktree.Filesystem.Root()
	pterm.Debug.Println("Root directory of Git repository:", gitRoot)

	status, err := Worktree.Status()
	if err != nil {
		fmt.Println("Failed to get status:", err)
		os.Exit(1)
	}

	// Check if there are staged changes
	hasStagedChanges := false
	hasUntracked := false
	for _, entry := range status {
		if entry.Staging != git.Untracked && entry.Staging != git.Unmodified {
			hasStagedChanges = true
			break
		} else if entry.Staging == git.Untracked {
			hasUntracked = true
		}
	}

	// Error out if nothing is staged
	if !hasStagedChanges && hasUntracked {
		pterm.Error.Println("nothing added to commit but untracked files present (use \"git add\" to track)")
		os.Exit(2)
	} else if !hasStagedChanges {
		pterm.Error.Println("nothing added to commit")
		os.Exit(2)
	}
}

func loadConfig() {
	// Set the file name of the configuration file
	viper.SetConfigName(".git-cc.yaml")
	// config file format
	viper.SetConfigType("yaml")
	// Add the path to look for the config file
	viper.AddConfigPath(gitRoot)
	// Optional. If you want to support environment variables, use this
	viper.AutomaticEnv()

	// Set Default Config Values
	viper.SetDefault("use_defaults", true)
	viper.SetDefault("custom_commit_types", []string{})
	viper.SetDefault("scopes", []string{})

	default_commit_types := []string{"feat", "fix", "build", "chore", "ci", "docs", "refactor", "test"}

	// Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		pterm.Debug.Printfln("Error reading config file: %s \n", err)
	}

	use_defaults := viper.GetBool("use_defaults")
	if use_defaults {
		commitTypes = append(default_commit_types, viper.GetStringSlice("custom_commit_types")...)
		if len(viper.GetStringSlice("scopes")) > 0 {
			scopes = append([]string{"none"}, viper.GetStringSlice("scopes")...)
		}
	} else {
		commitTypes = viper.GetStringSlice("custom_commit_types")
		scopes = viper.GetStringSlice("scopes")
	}
	// dedup slices just in case
	commitTypes = removeDuplicateStr(commitTypes)
	scopes = removeDuplicateStr(scopes)
}

func openGitRepo() (*git.Repository, error) {
	// Validate the current directory is a git repository
	cwd, err := os.Getwd()
	if err != nil {
		pterm.Fatal.Println("Error getting current working directory:", err)
	}

	// Open the Git repository at the current working directory
	repo, err := git.PlainOpenWithOptions(cwd, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, fmt.Errorf("not a git repository (or any of the parent directories): .git")
	}

	return repo, nil
}

func parseFlags() {
	var showVersion bool

	// Define a flag for version
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	// Parse command-line arguments
	flag.Parse()

	// show version info and exist
	if showVersion {
		fmt.Printf("version: %s, commit: %s, built at %s\n", version, commit, date)
		os.Exit(0)
	}
}

func promptForCommit(commitTypes []string) (string, error) {
	var commitMessage strings.Builder
	var data CommitPromptData
	swapFile := ".git-cc.swp"

	// Load from swap file if it exists
	if _, err := os.Stat(swapFile); err == nil {
		content, err := os.ReadFile(swapFile)
		if err == nil {
			json.Unmarshal(content, &data)
			pterm.Warning.Println("Restored previous session from .git-cc.swp")
		}
	}

	// Prompt for commit type
	defaultType := data.CommitType
	if defaultType == "" {
		defaultType = "Commit Type"
	}
	data.CommitType, _ = pterm.DefaultInteractiveSelect.
		WithOptions(commitTypes).
		WithDefaultText("Commit Type").
		WithDefaultOption(defaultType).
		WithMaxHeight(20).
		Show()

	// Prompt for scope
	if len(scopes) > 0 {
		defaultScope := data.Scope
		if defaultScope == "" {
			defaultScope = "none"
		}
		data.Scope, _ = pterm.DefaultInteractiveSelect.
			WithOptions(scopes).
			WithDefaultText("Scope").
			WithDefaultOption(defaultScope).
			WithMaxHeight(10).
			Show()
	} else {
		data.Scope, _ = pterm.DefaultInteractiveTextInput.
			WithDefaultText("Scope (optional)").
			WithDefaultValue(data.Scope).
			Show()
	}

	// Short description
	data.ShortDescription, _ = pterm.DefaultInteractiveTextInput.
		WithDefaultText("Short Description").
		WithDefaultValue(data.ShortDescription).
		Show()

	// Long description
	data.LongDescription, _ = pterm.DefaultInteractiveTextInput.
		WithMultiLine().
		WithDefaultText("Long Description (optional)").
		WithDefaultValue(data.LongDescription).
		Show()
	data.LongDescription = strings.TrimSpace(data.LongDescription)

	// Breaking change
	data.BreakingChange, _ = pterm.DefaultInteractiveConfirm.
		WithDefaultText("Breaking Change").
		WithDefaultValue(data.BreakingChange).
		Show()

	if data.BreakingChange {
		data.BreakingChangeNote, _ = pterm.DefaultInteractiveTextInput.
			WithDefaultText("Breaking Change Note").
			WithDefaultValue(data.BreakingChangeNote).
			Show()
	}

	// Save current prompt state to swap file
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	_ = os.WriteFile(swapFile, jsonData, 0644)

	// Build commit message
	commitMessage.WriteString(data.CommitType)
	if len(data.Scope) > 0 && data.Scope != "none" {
		commitMessage.WriteString("(" + data.Scope + ")")
	}

	if data.BreakingChange {
		commitMessage.WriteString("!: " + data.ShortDescription)
	} else {
		commitMessage.WriteString(": " + data.ShortDescription)
	}

	if len(data.LongDescription) > 0 {
		commitMessage.WriteString("\n\n" + data.LongDescription)
	}

	if data.BreakingChange && len(data.BreakingChangeNote) > 0 {
		commitMessage.WriteString("\n\nBREAKING CHANGE: " + data.BreakingChangeNote)
	}

	return commitMessage.String(), nil
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func init() {
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		// Enable debug messages in PTerm.
		pterm.EnableDebugMessages()
	}

	// Parse argument flags
	parseFlags()

	// Validate we are running in a git repo and get status
	gitStatus()

	// load optional config file
	loadConfig()

}

func main() {
	// Prompt and build commit message
	commitMsg, _ := promptForCommit(commitTypes)

	// Create a temporary file
	f, err := os.CreateTemp("", "commitMessage")
	if err != nil {
		pterm.Fatal.Println(err)
	}
	defer os.Remove(f.Name()) // clean up

	if _, err := f.WriteString(commitMsg); err != nil {
		pterm.Fatal.Println(err)
	}
	if err := f.Close(); err != nil {
		pterm.Fatal.Println(err)
	}
	pterm.Debug.Println(commitMsg)
	pterm.Debug.Println("temp file: " + f.Name())

	// run git commit passing commit message, this ensures pre-commit hooks are run
	cmd := exec.Command("git", "commit", "-F", f.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	err = cmd.Run()
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(3)
	}

	// Clean up swap file after successful commit
	_ = os.Remove(".git-cc.swp")
}
