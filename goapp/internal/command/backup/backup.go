package backup

import (
	"clickhouse-tools/internal/helper"
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/pkg/archiver"
	"fmt"
	archiverLibrary "github.com/mholt/archiver/v3"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	backup        = "backup"
	shadow        = "shadow"
	metadata      = "metadata"
	tablesIdsDir  = "tables"
	TimeFormat    = "2006-01-02T15-04-05"
	incrementFile = "increment.txt"
)

type Tool struct {
	clickhouse *clickhouse.Client
	command    *cli.Command
	archiver   *archiver.Archiver
	paths      *Paths
	name       string
}

type Paths struct {
	base, shadow, archive string
}

func New(cliApp *cli.App, clickhouseClient *clickhouse.Client, archiver *archiver.Archiver) *Tool {
	return &Tool{
		clickhouse: clickhouseClient,
		archiver:   archiver,
		command: &cli.Command{
			Name:        "backup",
			Usage:       "Create new backup",
			UsageText:   "clickhouse-tools backup [-db, --database=<database>]",
			Description: "Create new backup",
			Flags: append(cliApp.Flags,
				&cli.StringFlag{
					Name:     "database",
					Aliases:  []string{"db"},
					Hidden:   false,
					Required: true,
				},
			),
		},
		paths: &Paths{
			base:   path.Join(clickhouse.DefaultDataPath, backup),
			shadow: path.Join(clickhouse.DefaultDataPath, shadow),
		},
	}
}

func (tool *Tool) GetCommand() *cli.Command {
	tool.command.Action = func(c *cli.Context) error {
		return tool.Backup(c.String("database"))
	}
	return tool.command
}

func (tool *Tool) Backup(database string) error {
	fmt.Println("Starting backup!")
	if err := tool.createPaths(); err != nil {
		return err
	}
	if err := tool.clickhouse.Connect(""); err != nil {
		return err
	}
	defer tool.clickhouse.CloseConnection()
	tool.name = fmt.Sprintf("%s_%s", database, time.Now().UTC().Format(TimeFormat))
	tool.paths.archive = strings.Join([]string{path.Join(tool.paths.base, tool.name), tool.archiver.GetExtension()}, ".")
	writer, err := tool.archiver.Create(tool.paths.archive)
	if err != nil {
		return err
	}
	defer func(writer archiverLibrary.Writer) {
		err := writer.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(writer)
	tables, err := tool.clickhouse.GetTables(database)
	if err != nil {
		return err
	}
	if err := tool.clickhouse.Freeze(database, tables); err != nil {
		return err
	}
	if err := tool.backupMetadata(writer, tables); err != nil {
		return err
	}
	if err := tool.backupShadow(writer); err != nil {
		return err
	}
	fmt.Printf("Successful finish backup '%s'!\n", tool.paths.archive)
	return nil
}

func (tool *Tool) createPaths() error {
	fmt.Print("Create backup path...")
	_, err := os.Stat(tool.paths.base)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	if err == nil || !os.IsNotExist(err) {
		helper.ColoredPrintln(helper.ColorYellow, "already exists!")
		return nil
	}
	if err := os.MkdirAll(tool.paths.base, os.ModePerm); err != nil {
		log.Errorf("can't create backup path: %v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (tool *Tool) backupMetadata(writer archiverLibrary.Writer, tables []clickhouse.Table) error {
	for _, table := range tables {
		filename := path.Join(metadata, path.Base(table.MetadataPath))
		tmpFile := path.Join("/tmp", path.Base(table.MetadataPath))
		if err := helper.CreateFile(tmpFile, table.Query); err != nil {
			return err
		}
		if err := tool.archiver.AddFile(
			writer,
			&archiver.File{
				Path: tmpFile,
				Name: filename,
				Info: nil,
			},
		); err != nil {
			return err
		}
		if err := os.Remove(tmpFile); err != nil {
			log.Errorf("%+v", err)
			return err
		}
		baseFileName := strings.Join([]string{table.Name, "uuid"}, ".")
		filename = path.Join(tablesIdsDir, baseFileName)
		tmpFile = path.Join("/tmp", baseFileName)
		if err := helper.CreateFile(tmpFile, table.UUID); err != nil {
			return err
		}
		if err := tool.archiver.AddFile(
			writer,
			&archiver.File{
				Path: tmpFile,
				Name: filename,
				Info: nil,
			},
		); err != nil {
			return err
		}
		if err := os.Remove(tmpFile); err != nil {
			log.Errorf("%+v", err)
			return err
		}
	}
	return nil
}

func (tool *Tool) backupShadow(writer archiverLibrary.Writer) error {
	increment, err := helper.ReadFile(path.Join(clickhouse.DefaultDataPath, shadow, incrementFile))
	if err != nil {
		return err
	}
	shadowPath := path.Join(clickhouse.DefaultDataPath, shadow, increment, "store")
	if err := filepath.Walk(shadowPath, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("%+v", err)
			return err
		}
		filename := path.Join("data", strings.Replace(filePath, shadowPath, "", 1))
		if fileInfo.IsDir() {
			return nil
		}
		if !fileInfo.Mode().IsRegular() {
			return nil
		}
		if err := tool.archiver.AddFile(
			writer,
			&archiver.File{
				Path: filePath,
				Name: filename,
				Info: fileInfo,
			},
		); err != nil {
			log.Errorf("%+v", err)
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if err = os.RemoveAll(path.Join(clickhouse.DefaultDataPath, shadow)); err != nil {
		log.Errorf("%+v", err)
		return err
	}
	return nil
}

func (tool *Tool) GetArchiveName() string {
	return strings.Join([]string{path.Join(tool.name), tool.archiver.GetExtension()}, ".")
}
