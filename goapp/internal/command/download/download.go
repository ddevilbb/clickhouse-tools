package download

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

type Tool struct {
	config  *config.Application
	command *cli.Command
}

func New(cliApp *cli.App, conf *config.Application) *Tool {
	return &Tool{
		config: conf,
		command: &cli.Command{
			Name:        "download",
			Usage:       "Download backup from remote storage",
			UsageText:   "clickhouse-tools download [-s, --storage=<storage>] <backup_name>",
			Description: "Download backup from remote storage",
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

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.download(c, c.Args().First(), c.String("storage"))
	}
	return tool.command
}

func (tool *Tool) download(c *cli.Context, backupName, storageName string) error {
	if backupName == "" {
		log.Errorf("%+v", errors.New("backup name must be defined"))
		cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
	}
	fmt.Println("Starting download backup!")
	storageObj, err := storage.InitStorage(tool.config, storageName)
	if err != nil {
		return err
	}
	if err := storageObj.Download(path.Join(clickhouse.DefaultDataPath, "backup", backupName), backupName); err != nil {
		return err
	}
	fmt.Println("Successful finish download backup!")
	return nil
}
