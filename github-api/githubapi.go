package githubapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const BASE_URL = "https://api.github.com"

var (
	EXCLUDE_REPOS = map[string]struct{}{"kickstart.nvim": {}}
	repos         = []Repo{}
)

type Repo struct {
	Name      string    `json:"name"`
	Private   bool      `json:"private"`
	UpdatedAt time.Time `json:"updated_at"`
	SSHURL    string    `json:"ssh_url"`
	Language  string    `json:"language"`
}

type commit struct {
	Commit struct {
		Committer struct {
			Date time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

func DidCommitToday(token string) bool {
	client := &http.Client{}

	err := retrievePublicRepos(client, token)
	if err != nil {
		log.Printf("Error retrieving public repos: %v\n", err)
	}

	today := date(time.Now())
	for _, repo := range repos {
		if diff := timeDiffInDays(today, repo.UpdatedAt); diff > 10 {
			continue
		}
		commits, err := retrieveCommits(client, repo, token)
		if err != nil {
			log.Printf("Error retrieving commits for repo %s: %v\n", repo.Name, err)
		}
		for _, commit := range commits {
			if diff := timeDiffInDays(today, commit.Commit.Committer.Date); diff == 0 {
				return true
			}
		}
	}
	return false
}

func retrievePublicRepos(client *http.Client, token string) error {
	const url = BASE_URL + "/user/repos"

	resp, err := doRequest(client, url, token)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var allRepos []Repo
	err = json.NewDecoder(resp.Body).Decode(&allRepos)
	if err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	repos = filterPublicRepos(allRepos)
	return nil
}

func retrieveCommits(client *http.Client, repo Repo, token string) ([]commit, error) {
	var url = fmt.Sprintf("%s/repos/aaronbittel/%s/commits", BASE_URL, repo.Name)

	resp, err := doRequest(client, url, token)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	var commits []commit
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return commits, nil
}

func doRequest(client *http.Client, url, token string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	resp, err := client.Do(req)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	return resp, err
}

func GetRepos() []Repo {
	return repos
}

func filterPublicRepos(repos []Repo) []Repo {
	var publicRepos []Repo
	for _, repo := range repos {
		if repo.Private {
			continue
		}
		if _, ok := EXCLUDE_REPOS[repo.Name]; ok {
			continue
		}

		publicRepos = append(publicRepos, repo)
	}
	return publicRepos
}

func timeDiffInDays(today, commit time.Time) int {
	commitDate := date(commit)
	return int(time.Duration(today.Sub(commitDate).Hours()) / 24)
}

func date(d time.Time) time.Time {
	year, month, day := d.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func RepoByName(repos []Repo, name string) (Repo, error) {
	for _, repo := range repos {
		if repo.Name == name {
			return repo, nil
		}
	}
	return Repo{}, fmt.Errorf("no repo with that name")
}
