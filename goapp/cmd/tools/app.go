package main

import (
	"clickhouse-tools/internal/command"
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/internal/service/logger"
	"log"
	"os"
)

func main() {
	mainConfig := config.New()
	logger.Init(mainConfig)
	tools := command.New(mainConfig)
	if err := tools.App.Run(os.Args); err != nil {
		log.Printf("%+v", err)
	}
}
