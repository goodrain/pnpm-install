package main

import (
	"log"
	"os"

	pnpminstall "github.com/goodrain/pnpm-install"
	"github.com/goodrain/pnpm-install/cmd/setup-symlinks/internal"
)

func main() {
	projectPath, set := os.LookupEnv("NODE_PROJECT_PATH")
	if !set {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}

	err := internal.Run(os.Args[0], projectPath, pnpminstall.NewLinkedModuleResolver(pnpminstall.NewLinker(os.TempDir())))
	if err != nil {
		log.Fatal(err)
	}
}
