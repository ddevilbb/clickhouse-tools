package cluster

import (
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/internal/service/config"
	"fmt"
	"github.com/urfave/cli/v2"
)

type Tool struct {
	conf       *config.Application
	command    *cli.Command
	clickhouse *clickhouse.Client
}

func New(cliApp *cli.App, conf *config.Application, clickhouse *clickhouse.Client) *Tool {
	return &Tool{
		conf: conf,
		command: &cli.Command{
			Name:        "clusters",
			Usage:       "Get clusters list",
			UsageText:   "clickhouse-tools clusters [-db, --database=<database>]",
			Description: "Get clusters list",
			Flags: append(cliApp.Flags,
				&cli.StringFlag{
					Name:     "database",
					Aliases:  []string{"db"},
					Hidden:   false,
					Required: true,
				},
			),
		},
		clickhouse: clickhouse,
	}
}

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.printClusters(c.String("database"))
	}
	return tool.command
}

func (tool *Tool) printClusters(database string) error {
	if err := tool.clickhouse.Connect(database); err != nil {
		return err
	}
	defer tool.clickhouse.CloseConnection()
	clusters, err := tool.clickhouse.GetClusters()
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		fmt.Println("no clusters found")
		return nil
	}
	for _, cluster := range clusters {
		fmt.Println(cluster)
	}
	return nil
}
