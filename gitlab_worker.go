package main

import (
	"github.com/xanzy/go-gitlab"
	"sync"
	"time"
)

type gitlabWorker struct {
	Client *GitlabClient
}

func (g gitlabWorker) doWork(wg *sync.WaitGroup, checker *Checker) {
	defer wg.Done()
	client := g.Client

	projectsChan := make(chan int)
	wait := client.processProjects(checker, projectsChan)
	listProjectOptions := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 20, Page: 1},
	}

	for {
		projects, resp, err := client.Projects.ListProjects(listProjectOptions)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, project := range projects {
			if checker.checkProjectNameBlacklisted(project.PathWithNamespace) {
				continue
			}
			projectsChan <- project.ID
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		listProjectOptions.Page = resp.NextPage
	}
	close(projectsChan)
	<-wait
}
