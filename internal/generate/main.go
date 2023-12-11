package main

import (
	"github.com/michaelangeloio/go-embed-python/pip"
)

func main() {
	err := pip.CreateEmbeddedPipPackagesForKnownPlatforms("requirements.txt", "./data/")
	if err != nil {
		panic(err)
	}
}
