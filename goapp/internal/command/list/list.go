package list

import (
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/internal/service/storage"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"regexp"
	"sort"
	"time"
)

const (
	remote = "remote"
)

type Tool struct {
	config  *config.Application
	command *cli.Command
	paths   *Paths
}

type Paths struct {
	local, remote string
}

type Backup struct {
	Name string
	Size int64
	Date time.Time
}

func New(cliApp *cli.App, conf *config.Application) *Tool {
	return &Tool{
		config: conf,
		command: &cli.Command{
			Name:        "list",
			Usage:       "Print backup list",
			UsageText:   "clickhouse-tools list [-s, --storage=<storage>] [local|remote]",
			Description: "Print backup list. Default: local",
			Flags: append(cliApp.Flags,
				&cli.StringFlag{
					Name:     "storage",
					Aliases:  []string{"s"},
					Hidden:   false,
					Required: false,
				},
			),
		},
		paths: &Paths{
			local:  path.Join(clickhouse.DefaultDataPath, "backup"),
			remote: fmt.Sprintf("%s:%s", conf.Rsync.Host, conf.Rsync.RemotePath),
		},
	}
}

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.list(c, c.Args().First(), c.String("storage"))
	}
	return tool.command
}

func (tool *Tool) list(c *cli.Context, direction, storageName string) error {
	switch direction {
	case remote:
		if storageName == "" {
			log.Errorf("%+v", errors.New("storage must be defined for remote list"))
			cli.ShowCommandHelpAndExit(c, c.Command.Name, 1)
		}
		return tool.printRemoteBackupList(storageName)
	default:
		return tool.printLocalBackupList()
	}
}

func (tool *Tool) printRemoteBackupList(storageName string) error {
	storageObj, err := storage.InitStorage(tool.config, storageName)
	if err != nil {
		return err
	}
	listString, err := storageObj.GetBackupListString()
	if err != nil {
		return err
	}
	var backupList []Backup
	var re = regexp.MustCompile(`(?m)(?P<Date>\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}) (?P<Name>[a-z_]*\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}[a-z0-9.]+)$`)
	for _, match := range re.FindAllStringSubmatch(listString, -1) {
		date, _ := time.Parse("2006/01/02 15:04:05", match[1])
		backupList = append(backupList, Backup{
			Name: match[2],
			Date: date,
		})
	}
	tool.printBackupList(backupList, false)
	return nil
}

func (tool *Tool) printLocalBackupList() error {
	dir, err := os.Open(tool.paths.local)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	names, err := dir.Readdirnames(0)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	var backupList []Backup
	for _, name := range names {
		info, err := os.Stat(path.Join(tool.paths.local, name))
		if err != nil {
			continue
		}
		backupList = append(backupList, Backup{
			Name: info.Name(),
			Date: info.ModTime(),
		})
	}
	tool.printBackupList(backupList, false)
	return nil
}

func (tool *Tool) printBackupList(backupList []Backup, printSize bool) {
	if len(backupList) == 0 {
		fmt.Println("no backups found")
		return
	}
	sort.SliceStable(backupList, func(i, j int) bool {
		return backupList[i].Date.After(backupList[j].Date)
	})
	if printSize {
		for _, backup := range backupList {
			fmt.Printf("- '%s'\t%s\t(created at %s)\n", backup.Name, backup.Size, backup.Date.Format("02-01-2006 15:04:05"))
		}
	} else {
		for _, backup := range backupList {
			fmt.Printf("- '%s'\t(created at %s)\n", backup.Name, backup.Date.Format("02-01-2006 15:04:05"))
		}
	}
}
