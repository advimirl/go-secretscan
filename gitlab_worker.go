package main

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type gitlabWorker struct {
	Client *GitlabClient
}

func (g gitlabWorker) doWork(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	client := g.Client

	projectsChan := make(chan int)
	wait := client.createTasksAndWait(ctx, projectsChan)
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
			if checkProjectNameBlacklisted(ctx, project.PathWithNamespace) {
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
	logrus.Info("Generating reports....")
	storage := getContextStorage(ctx)
	for projectName, messages := range getContextMessageStore(ctx).store {
		report, err := createReport(messages, storage)
		if err != nil {
			logrus.Error(err)
			continue
		}
		if ok, err := saveAndSendReport(report, projectName, storage); !ok || err != nil {
			logrus.Error(err)
			continue
		}
		logrus.Infof("Generation done for %s", projectName)
	}
}
