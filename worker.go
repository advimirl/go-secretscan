package main

import (
	"github.com/doublestraus/go-bitbucket"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"sync"
	"time"
)

type Worker interface {
	doWork(wg *sync.WaitGroup, checker *Checker)
}

const (
	GitlabWorkerType    = "gitlab"
	BitbucketWorkerType = "bitbucket"
)

func createWorker(accessToken AccessToken, folder string, forceCreation bool) Worker {
	scanSession := createSession(folder, accessToken)
	if scanSession.exists() && !forceCreation {
		logrus.Printf("[%s] - [%s] - Scanning session already exists.\nIf you DONT want to continue the previous scan use --force argument to renew session.\nScanning will resume in 15 seconds", accessToken.WorkerType, accessToken.URL)
		time.Sleep(15 * time.Second)
	}
	scanSession.init(forceCreation)
	switch accessToken.WorkerType {
	case GitlabWorkerType:
		return createGitlabWorker(accessToken, scanSession)
	case BitbucketWorkerType:
		return createBitbucketWorker(accessToken, scanSession)
	default:
		panic("Choose implemented worker")
	}
}

func createGitlabWorker(accessToken AccessToken, scanSession *Session) gitlabWorker {
	if accessToken.Token == "" {
		panic("Cannot create worker without token")
	}
	if accessToken.URL == "" {
		panic("Cannot create worker without url")
	}

	client, err := gitlab.NewClient(accessToken.Token, gitlab.WithBaseURL(accessToken.URL), gitlabIgnoreAuthority())
	if err != nil {
		panic(err)
	}
	client.UserAgent = Name
	// To check availability of gitlab instance
	_, resp, err := client.Version.GetVersion()
	if resp.StatusCode == 403 {
		panic("Invalid access token")
	}
	if err != nil {
		panic(err)
	}

	gClient := &GitlabClient{
		client,
		&accessToken,
		scanSession,
	}

	return gitlabWorker{Client: gClient, session: scanSession}
}

func createBitbucketWorker(accessToken AccessToken, scanSession *Session) bitbucketWorker {
	client := bitbucket.New(accessToken.Token, accessToken.URL)
	bClient := &BitbucketClient{
		client,
		&accessToken,
		scanSession,
	}
	return bitbucketWorker{client: bClient}
}
