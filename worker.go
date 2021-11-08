package main

import (
	"github.com/doublestraus/go-bitbucket"
	"github.com/xanzy/go-gitlab"
	"sync"
)

type Worker interface {
	doWork(wg *sync.WaitGroup, checker *Checker)
}

const (
	GitlabWorkerType    = "gitlab"
	BitbucketWorkerType = "bitbucket"
)

func createWorker(accessToken AccessToken) Worker {
	switch accessToken.WorkerType {
	case GitlabWorkerType:
		return createGitlabWorker(accessToken)
	case BitbucketWorkerType:
		return createBitbucketWorker(accessToken)
	default:
		panic("Choose implemented worker")
	}
}

func createGitlabWorker(accessToken AccessToken) gitlabWorker {
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
	}

	return gitlabWorker{Client: gClient}
}

func createBitbucketWorker(accessToken AccessToken) bitbucketWorker {
	client := bitbucket.New(accessToken.Token, accessToken.URL)
	bClient := &BitbucketClient{
		client,
		&accessToken,
	}
	return bitbucketWorker{bClient}
}
