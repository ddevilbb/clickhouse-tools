package upload

import (
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/internal/service/storage"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"path"
)

const (
	upload = "upload"
)

type Upload struct {
	config  *config.Application
	command *cli.Command
}

func New(cliApp *cli.App, conf *config.Application) *Upload {
	return &Upload{
		config: conf,
		command: &cli.Command{
			Name:        "upload",
			Usage:       "Upload backup to remote storage",
			UsageText:   "clickhouse-tools upload [-s, --storage=<storage>] <backup_name>",
			Description: "Upload backup to remote storage",
			Flags: append(cliApp.Flags,
				&cli.StringFlag{
					Name:     "storage",
					Aliases:  []string{"s"},
					Hidden:   false,
					Required: true,
				},
			),
		},
	}
}

func (tool *Upload) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.Upload(c, c.Args().First(), c.String("storage"))
	}
	return tool.command
}

func (tool *Upload) Upload(c *cli.Context, backupName, storageName string) error {
	if backupName == "" {
		log.Errorf("%+v", errors.New("backup name must be defined"))
		cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
	}
	fmt.Println("Starting upload backup!")
	storageObj, err := storage.InitStorage(tool.config, storageName)
	if err != nil {
		return err
	}
	if err := storageObj.Upload(path.Join(clickhouse.DefaultDataPath, "backup", backupName)); err != nil {
		return err
	}
	fmt.Println("Successful finish upload backup!")
	return nil
}
