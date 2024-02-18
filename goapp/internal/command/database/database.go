package database

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
			Name:        "databases",
			Usage:       "Get databases list",
			UsageText:   "clickhouse-tools databases",
			Description: "Get databases list",
			Flags:       cliApp.Flags,
		},
		clickhouse: clickhouse,
	}
}

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.printDatabases()
	}
	return tool.command
}

func (tool *Tool) printDatabases() error {
	if err := tool.clickhouse.Connect(""); err != nil {
		return err
	}
	defer tool.clickhouse.CloseConnection()
	clusters, err := tool.clickhouse.GetDatabases()
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		fmt.Println("no databases found")
		return nil
	}
	for _, cluster := range clusters {
		fmt.Println(cluster)
	}
	return nil
}
