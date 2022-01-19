package main

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	ctxKeyOptions        = ctxKey("options")
	ctxKeyStorage        = ctxKey("storage")
	ctxKeyReportMessages = ctxKey("report-messages")
	ctxKeyScanSession    = ctxKey("scan-session")
)

func main() {
	options, err := parseOptions()
	if err != nil {
		logrus.Panicf("Cannot parse options: %s", err)
	}
	if options.Silent && !options.Debug {
		logrus.SetLevel(logrus.ErrorLevel)
	} else if options.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	storage := createStorage(options)
	var wgWorkers sync.WaitGroup

	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxKeyOptions, options)
	ctx = context.WithValue(ctx, ctxKeyStorage, storage)
	for _, access := range storage.getAccessTokens() {
		worker, wctx := createWorker(ctx, access)
		wctx = context.WithValue(wctx, ctxKeyReportMessages, newMessageStore())
		wgWorkers.Add(1)
		go worker.doWork(wctx, &wgWorkers)
	}
	wgWorkers.Wait()
	logrus.Println("Done.")
}
