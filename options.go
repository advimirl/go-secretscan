package main

import (
	"flag"
)

const Name = "go-secretscan"

type Options struct {
	Silent         bool
	Debug          bool
	PathChecks     bool
	ConfigPath     string
	ReportsDir     string
	Dodjo          DodjoOpts
	SignaturesPath string
}

type DodjoOpts struct {
	Url     string
	Token   string
	Product string
}

func parseOptions() (*Options, error) {
	options := new(Options)

	flag.BoolVar(&options.Silent, "silent", false, "Suppress all output except for errors")
	flag.BoolVar(&options.Debug, "debug", false, "Print debugging information")
	flag.BoolVar(&options.PathChecks, "path-checks", true, "Set to false to disable checking of filepaths, i.e. just match regex patterns of file contents")
	flag.StringVar(&options.ConfigPath, "config", "config/config.yaml", "Path to config.yaml file")
	flag.StringVar(&options.ReportsDir, "out", "reports", "Directory for report store")
	flag.StringVar(&options.Dodjo.Url, "dd-url", "", "Defect Dodjo url")
	flag.StringVar(&options.Dodjo.Token, "dd-token", "", "Defect Dodjo API token")
	flag.StringVar(&options.Dodjo.Product, "dd-product", "", "Defect Dodjo product")
	flag.StringVar(&options.SignaturesPath, "signature", "config/signatures.yaml", "Path to signatures.yml file")
	flag.Parse()
	return options, nil
}
