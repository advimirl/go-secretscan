package main

import (
	"github.com/doublestraus/go-bitbucket"
	"sync"
	"time"
)

type bitbucketWorker struct {
	client *BitbucketClient
}

func (b bitbucketWorker) doWork(wg *sync.WaitGroup, checker *Checker) {
	defer wg.Done()
	client := b.client

	projectsChan := make(chan string)
	wait := client.ProcessProjects(checker, projectsChan)
	pagination := bitbucket.DefaultPagination()
	filter := &bitbucket.ProjectsFilter{}

	for {
		projects, err := client.ListProjects(pagination, filter)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, project := range projects {
			if checker.checkProjectNameBlacklisted(project.Name) {
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
