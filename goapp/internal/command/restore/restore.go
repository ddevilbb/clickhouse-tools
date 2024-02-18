package restore

import (
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/pkg/archiver"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"path"
	"strings"
)

const (
	backup = "backup"
)

type Tool struct {
	config     *config.Application
	command    *cli.Command
	clickhouse *clickhouse.Client
	archiver   *archiver.Archiver
	paths      *Paths
}

type Paths struct {
	base string
}

func New(cliApp *cli.App, conf *config.Application, clickhouseClient *clickhouse.Client, archiver *archiver.Archiver) *Tool {
	basePath := path.Join(clickhouse.DefaultDataPath, backup)
	return &Tool{
		config:     conf,
		clickhouse: clickhouseClient,
		archiver:   archiver,
		command: &cli.Command{
			Name:        "restore",
			Usage:       "Restore backup",
			UsageText:   "clickhouse-tools restore [-c, --cluster=<cluster>] [-db, --database=<database>] <backup_name>",
			Description: "Restore backup",
			Flags: append(cliApp.Flags,
				&cli.StringFlag{
					Name:     "cluster",
					Aliases:  []string{"c"},
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
		paths: &Paths{
			base: basePath,
		},
	}
}

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.restore(c, c.Args().First(), c.String("cluster"), c.String("database"))
	}
	return tool.command
}

func (tool *Tool) restore(c *cli.Context, backupName, cluster, database string) error {
	inCluster := false
	if err := tool.clickhouse.Connect(""); err != nil {
		return err
	}
	defer tool.clickhouse.CloseConnection()
	clusters, err := tool.clickhouse.GetClusters()
	if err != nil {
		return err
	}
	if len(clusters) > 0 {
		inCluster = true
	}
	if backupName == "" {
		log.Errorf("%+v", errors.New("backup name must be defined"))
		cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
	}
	srcPath := path.Join(tool.paths.base, backupName)
	dstPath := strings.TrimSuffix(srcPath, "."+tool.archiver.GetExtension())
	if err := tool.archiver.Unarchive(srcPath, dstPath); err != nil {
		return err
	}
	if err := tool.clickhouse.DropAllData(database, cluster, inCluster); err != nil {
		return err
	}
	if err := tool.clickhouse.RestoreTablesSchemas(database, dstPath, cluster, inCluster); err != nil {
		return err
	}
	if err := tool.clickhouse.RestoreTablesData(database, path.Join(dstPath, "data")); err != nil {
		return err
	}
	return nil
}
