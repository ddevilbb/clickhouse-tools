package task

import (
	"clickhouse-tools/internal/command/backup"
	"clickhouse-tools/internal/command/upload"
	"github.com/urfave/cli/v2"
)

type Tool struct {
	backupTool *backup.Tool
	uploadTool *upload.Upload
	command    *cli.Command
}

func New(cliApp *cli.App, backupTool *backup.Tool, uploadTool *upload.Upload) *Tool {
	return &Tool{
		backupTool: backupTool,
		uploadTool: uploadTool,
		command: &cli.Command{
			Name:        "task",
			Usage:       "Run backup task",
			UsageText:   "clickhouse-tools task [-s, --storage=<storage>] [-db, --database=<database>]",
			Description: "Create new backup and upload it",
			Flags: append(cliApp.Flags,
				&cli.StringFlag{
					Name:     "storage",
					Aliases:  []string{"s"},
					Hidden:   false,
					Required: true,
				},
				&cli.StringFlag{
					Name:     "database",
					Aliases:  []string{"db"},
					Hidden:   false,
					Required: true,
				},
			),
		},
	}
}

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.runTask(c)
	}
	return tool.command
}

func (tool *Tool) runTask(c *cli.Context) error {
	if err := tool.backupTool.Backup(c.String("database")); err != nil {
		return err
	}
	if err := tool.uploadTool.Upload(c, tool.backupTool.GetArchiveName(), c.String("storage")); err != nil {
		return err
	}
	return nil
}
