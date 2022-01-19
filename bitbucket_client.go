package main

import (
	"context"
	"sync"
	"time"

	"github.com/doublestraus/go-bitbucket"
	"github.com/sirupsen/logrus"
)

type BitbucketClient struct {
	*bitbucket.Client
	accessToken *AccessToken
}

func (b *BitbucketClient) ProcessProjects(ctx context.Context, projectsChan <-chan string) (wait <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		var pg sync.WaitGroup
		for projectName := range projectsChan {
			pg.Add(1)
			go b.ProcessProject(ctx, &pg, projectName)
		}
		pg.Wait()
		close(ch)
	}()
	return ch
}

func (b *BitbucketClient) ProcessProject(ctx context.Context, wg *sync.WaitGroup, projectName string) {
	defer wg.Done()
	filter := &bitbucket.ProjectReposFilter{}
	for {
		pagination := bitbucket.DefaultPagination()
		repositories, err := b.GetProjectRepos(projectName, pagination, filter)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, repo := range repositories {
			b.processRepo(ctx, repo)
		}
		if pagination.IsLastPage {
			break
		}
		pagination.Start = pagination.NextPageStart
		break
	}
}

func (b *BitbucketClient) processRepo(ctx context.Context, repository *bitbucket.Repository) {
	pagination := bitbucket.DefaultPagination()
	filter := &bitbucket.ProjectReposFileFilter{}
	filesToProcess := make([]MatchFile, 0)
	optMonthToCheckFrom := getContextOptions(ctx).FromMonth
	if !b.repoActive(repository, optMonthToCheckFrom) {
		return
	}
	session := getContextScanSession(ctx)
	for {
		if session.check(repository.Project.Key + "/" + repository.Slug) {
			logrus.Debugf("%s has already been scanned.", repository.Project.Key+"/"+repository.Slug)
			time.Sleep(5 * time.Second)
			continue
		}
		files, err := b.GetProjectsReposFiles(repository.Project.Name, repository.Slug, pagination, filter)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, file := range files {
			if checkFileExtBlacklisted(ctx, file) {
				continue
			}
			if checkFilenameBlacklisted(ctx, file) {
				continue
			}
			fRaw, err := b.GetProjectsReposFileRaw(repository.Project.Name, repository.Slug, file)
			if err != nil {
				panic(err)
			}
			filesToProcess = append(filesToProcess, newMatchFile(file, fRaw, nil))
		}
		if pagination.IsLastPage {
			break
		}
		pagination.Start = pagination.NextPageStart
	}
	b.processMatch(ctx, filesToProcess, repository)
}

func (b *BitbucketClient) repoActive(repository *bitbucket.Repository, monthToCheck int) bool {
	pagination := bitbucket.DefaultPagination()
	commits, _ := b.GetProjectsReposCommits(repository.Project.Key, repository.Slug, pagination)
	t := time.Now().AddDate(0, -monthToCheck, 0)
	for _, commit := range commits {
		if commit.CommitterTimestamp.Time().After(t) {
			return true
		}
	}
	return false
}

func (b *BitbucketClient) processMatch(ctx context.Context, files []MatchFile, repository *bitbucket.Repository) {
	messages := getContextMessageStore(ctx).getMessages(repository.Name)
	for _, file := range files {
		if record, reason, matched := getMatch(ctx, file); matched {
			msg := &Message{
				ProjectID:          repository.Project.Id,
				ProjectName:        repository.Name,
				MatchURL:           repository.Links.Self[0].Href + "/" + file.Path,
				Path:               file.Path,
				Filename:           file.Filename,
				MatchType:          record.MatchType,
				MatchName:          reason,
				RawMatchContent:    record.MatchedContent,
				Confidence:         record.Confidence,
				MatchedLineNumbers: record.MatchedLineNumbers,
				commitInfo:         &CommitInfo{},
			}
			messages = append(messages, msg)
		}
	}
	session := getContextScanSession(ctx)
	session.add(repository.Project.Key + "/" + repository.Slug)
}
