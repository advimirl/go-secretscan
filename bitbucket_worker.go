package main

import (
	"context"
	"sync"
	"time"

	"github.com/doublestraus/go-bitbucket"
)

type bitbucketWorker struct {
	client *BitbucketClient
}

func (b bitbucketWorker) doWork(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	client := b.client

	projectsChan := make(chan string)
	wait := client.ProcessProjects(ctx, projectsChan)
	pagination := bitbucket.DefaultPagination()
	filter := &bitbucket.ProjectsFilter{}

	for {
		projects, err := client.ListProjects(pagination, filter)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, project := range projects {
			if checkProjectNameBlacklisted(ctx, project.Name) {
				continue
			}
			projectsChan <- project.Key
		}
		if pagination.IsLastPage {
			break
		}
		pagination.Start = pagination.NextPageStart
	}

	close(projectsChan)
	<-wait
}
