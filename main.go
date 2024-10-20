package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	githubapi "github.com/aaronbittel/github-commit-checker/github-api"
	"github.com/joho/godotenv"
)

const baseDir = "projects"

const (
	nicePromt = "Nice you did commit today"
)

func main() {
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable path: %v", err)
	}

	execDir := filepath.Dir(executablePath)

	err = godotenv.Load(filepath.Join(execDir, ".env"))
	if err != nil {
		log.Fatalf("could not load .env file: %v", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GitHub token not found in .env file")
	}

	var selectedRepo githubapi.Repo

	//TODO: Make this better
	if githubapi.DidCommitToday(token) {
		fmt.Println(nicePromt)
		return
	}

	repos := githubapi.GetRepos()

	selectedRepo = getSelectedRepo(repos)

	if checkDirectoryExists(selectedRepo.Name) {
		targetDir := getTargetDir(selectedRepo.Name)
		runTmuxSessionizerScript(targetDir)
		return
	}

	if !promptToClone(selectedRepo.Name) {
		os.Exit(0)
	}

	targetDir := targetDirPrompt()

	repoPath := filepath.Join(targetDir, selectedRepo.Name)
	err = cloneRepo(selectedRepo.SSHURL, repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error cloning repo: %v", err)
		os.Exit(1)
	}

	runTmuxSessionizerScript(repoPath)
}

func cloneRepo(repoURL, targetDir string) error {
	cmd := exec.Command("git", "clone", repoURL, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run() // Execute the command
}

// Run the bash script with the selected repo as an argument
func runTmuxSessionizerScript(repoPath string) {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting home directory")
		os.Exit(1)
	}

	scriptPath := filepath.Join(home, ".local", "bin", "tmux-sessionizer")

	// Prepare the command to execute the script and pass repo path as an argument
	cmd := exec.Command("/bin/bash", scriptPath, repoPath)

	// Run the command and capture output or errors
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatalf("Error running script: %v\n", err)
	}
}

func checkDirectoryExists(repoName string) bool {
	projectPaths, err := getProjectPaths()
	if err != nil {
		log.Fatal("")
	}
	for _, path := range projectPaths {
		dirs, err := os.ReadDir(path)
		if err != nil {
			log.Fatal("error reading dirs")
		}

		for _, dir := range dirs {
			if dir.Name() == repoName {
				return true
			}
		}
	}
	return false
}

func getProjectPaths() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projectSubdirs := []string{
		baseDir,
		filepath.Join(baseDir, "golang"),
		filepath.Join(baseDir, "rust"),
	}

	var projectPaths []string
	for _, subdir := range projectSubdirs {
		projectPaths = append(projectPaths, filepath.Join(homeDir, subdir))
	}

	return projectPaths, nil
}

func getSelectedRepo(repos []githubapi.Repo) githubapi.Repo {
	fmt.Println()
	fmt.Println("Select a repo:")
	for i, repo := range repos {
		fmt.Printf("%d: %s [%s]\n", i+1, repo.Name, repo.Language)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println()
		fmt.Println("Please enter a repo id: ")
		fmt.Print("> ")
		scanner.Scan()
		userInput := scanner.Text()
		repoId, err := strconv.Atoi(userInput)

		if err != nil {
			fmt.Fprintln(os.Stderr, "Please enter a number")
			continue
		}

		if repoId-1 < 0 || repoId-1 >= len(repos) {
			fmt.Fprintf(os.Stderr, "Please enter a number between 1 and %d\n", len(repos))
			continue
		}

		return repos[repoId-1]
	}
}

func promptToClone(repoName string) bool {
	fmt.Println()
	fmt.Printf("The repo (%s) currently does not exist locally.\n", repoName)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("Do you want to clone it? [Y/n] ")
		scanner.Scan()
		userInput := strings.ToLower(scanner.Text())
		switch userInput {
		case "y", "yes", "ye":
			return true
		case "n", "no":
			return false
		}
	}
}

func getTargetDir(repoName string) string {
	projectPaths, err := getProjectPaths()
	if err != nil {
		log.Fatal("could not load project paths")
	}

	for _, path := range projectPaths {
		dirs, err := os.ReadDir(path)
		if err != nil {
			log.Fatal("could not read projects dirs")
		}
		for _, dir := range dirs {
			if dir.Name() == repoName {
				return filepath.Join(path, repoName)
			}
		}
	}

	return ""
}

func targetDirPrompt() string {
	projectPaths, err := getProjectPaths()
	if err != nil {
		log.Fatal("could not load project paths")
	}

	fmt.Println()
	for i, path := range projectPaths {
		fmt.Printf("%d: %s\n", i+1, path)
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println()
		fmt.Println("In which project directory?")
		fmt.Print("> ")
		scanner.Scan()
		userInput := strings.ToLower(scanner.Text())

		dirId, err := strconv.Atoi(userInput)
		if err != nil {
			fmt.Println("please enter a number")
			continue
		}

		if dirId-1 < 0 || dirId-1 >= len(projectPaths) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(projectPaths))
			continue
		}

		fmt.Println()
		return projectPaths[dirId-1]
	}
}
