package command

import (
	"clickhouse-tools/internal/command/backup"
	"clickhouse-tools/internal/command/cluster"
	"clickhouse-tools/internal/command/database"
	"clickhouse-tools/internal/command/download"
	"clickhouse-tools/internal/command/list"
	"clickhouse-tools/internal/command/restore"
	"clickhouse-tools/internal/command/task"
	"clickhouse-tools/internal/command/upload"
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/pkg/archiver"
	"github.com/urfave/cli/v2"
)

const (
	version = "0.0.1"
)

type Tools struct {
	App *cli.App
}

func New(conf *config.Application) *Tools {
	Clickhouse := clickhouse.New(conf.Clickhouse)
	Archiver := archiver.New(conf.Archiver)
	cliApp := &cli.App{
		Name:        "clickhouse-tools",
		Usage:       "Tool for backup clickhouse",
		UsageText:   "clickhouse-tools <command>",
		Description: "Run as 'root' or 'clickhouse' user",
		Version:     version,
		Flags:       []cli.Flag{},
	}
	backupTool := backup.New(cliApp, Clickhouse, Archiver)
	uploadTool := upload.New(cliApp, conf)
	listTool := list.New(cliApp, conf)
	downloadTool := download.New(cliApp, conf)
	restoreTool := restore.New(cliApp, conf, Clickhouse, Archiver)
	clusterTool := cluster.New(cliApp, conf, Clickhouse)
	taskTool := task.New(cliApp, backupTool, uploadTool)
	databaseTool := database.New(cliApp, conf, Clickhouse)
	cliApp.Commands = []*cli.Command{
		backupTool.GetCommand(),
		uploadTool.GetCommand(),
		listTool.GetCommand(),
		downloadTool.GetCommand(),
		restoreTool.GetCommand(),
		clusterTool.GetCommand(),
		taskTool.GetCommand(),
		databaseTool.GetCommand(),
	}
	return &Tools{
		App: cliApp,
	}
}
