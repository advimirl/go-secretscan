package main

import (
	"github.com/sirupsen/logrus"
	"sync"
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

	storage := create(options)
	var wgWorkers sync.WaitGroup

	checker := Checker{storage}
	workers := make([]Worker, 0)
	for _, access := range storage.getAccessTokens() {
		worker := createWorker(access, options.ReportsDir, options.ForceCreation)
		workers = append(workers, worker)
	}
	for _, worker := range workers {
		wgWorkers.Add(1)
		go worker.doWork(&wgWorkers, &checker)
	}
	wgWorkers.Wait()
}
