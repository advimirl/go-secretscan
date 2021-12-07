package main

import (
	"github.com/doublestraus/go-bitbucket"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type BitbucketClient struct {
	*bitbucket.Client
	accessToken *AccessToken
	session *Session
}

func (b *BitbucketClient) ProcessProjects(checker *Checker, projectsChan <-chan string) (wait <-chan struct{}) {
	ch := make(chan struct{})
	go func() {
		var pg sync.WaitGroup
		for projectName := range projectsChan {
			pg.Add(1)
			go b.ProcessProject(&pg, projectName, checker)
		}
		pg.Wait()
		close(ch)
	}()
	return ch
}

func (b *BitbucketClient) ProcessProject(wg *sync.WaitGroup, projectName string, checker *Checker) {
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
			b.processRepo(checker, repo)
		}
		if pagination.IsLastPage {
			break
		}
		pagination.Start = pagination.NextPageStart
		break
	}
}

func (b *BitbucketClient) processRepo(checker *Checker, repository *bitbucket.Repository) {
	pagination := bitbucket.DefaultPagination()
	filter := &bitbucket.ProjectReposFileFilter{}
	filesToProcess := make([]MatchFile, 0)
	if !b.repoActive(repository, 1) {
		return
	}
	for {
		if b.session.check(repository.Project.Key + "/" + repository.Slug) {
			logrus.Debugf("%s has already been scanned.", repository.Project.Key + "/" + repository.Slug)
			time.Sleep(5 * time.Second)
			continue
		}
		files, err := b.GetProjectsReposFiles(repository.Project.Name, repository.Slug, pagination, filter)
		if isTimeout(err) {
			time.Sleep(5 * time.Second)
			continue
		}
		for _, file := range files {
			if checker.checkFileExtBlacklisted(file) {
				continue
			}
			if checker.checkFilenameBlacklisted(file) {
				continue
			}
			fRaw, err := b.GetProjectsReposFileRaw(repository.Project.Name, repository.Slug, file)
			if err != nil {
				panic(err)
			}
			filesToProcess = append(filesToProcess, newMatchFile(file, fRaw, nil, checker))
		}
		if pagination.IsLastPage {
			break
		}
		pagination.Start = pagination.NextPageStart
	}
	b.processMatch(filesToProcess, repository, checker.storage)
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

func (b *BitbucketClient) processMatch(files []MatchFile, repository *bitbucket.Repository, storage *Storage) {
	var (
		ok       = false
		records  []MatchRecord
		messages []Message
	)
	for _, file := range files {
		if record, reason, matched := getMatch(file); matched {
			records = append(records, record)
			ok = true
			msg := Message{
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

	if ok {
		report, err := createReport(messages, storage)
		if err != nil {
			logrus.Print(err)
		}
		if ok, err := saveAndSendReport(report, repository.Name, storage); !ok || err != nil {
			logrus.Println(err)
		}
	}
	b.session.add(repository.Project.Key + "/" + repository.Slug)
}
