package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/doublestraus/go-bitbucket"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type Worker interface {
	doWork(ctx context.Context, wg *sync.WaitGroup)
}

const (
	GitlabWorkerType    = "gitlab"
	BitbucketWorkerType = "bitbucket"
)

func createWorker(ctx context.Context, accessToken AccessToken) (Worker, context.Context) {
	options := getContextOptions(ctx)
	scanSession := createSession(options.ReportsDir, accessToken)
	if scanSession.exists() && !options.ForceCreation {
		logrus.Printf("[%s] - [%s] - Scanning session already exists.\nIf you DONT want to continue the previous scan use --force argument to renew session.\nScanning will resume in 15 seconds", accessToken.WorkerType, accessToken.URL)
		time.Sleep(15 * time.Second)
	}
	scanSession.init(options.ForceCreation)
	wctx := context.WithValue(ctx, ctxKeyScanSession, scanSession)
	switch accessToken.WorkerType {
	case GitlabWorkerType:
		return createGitlabWorker(accessToken), wctx
	case BitbucketWorkerType:
		return createBitbucketWorker(accessToken), wctx
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
		logrus.Error("Invalid access token")
		os.Exit(1)
	}
	if err != nil {
		logrus.Error(err)
		os.Exit(1)
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
	return bitbucketWorker{client: bClient}
}
