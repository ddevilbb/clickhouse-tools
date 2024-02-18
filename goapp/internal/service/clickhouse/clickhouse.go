package clickhouse

import (
	"clickhouse-tools/internal/helper"
	"fmt"
	clickhouseGo "github.com/ClickHouse/clickhouse-go"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const (
	DefaultDataPath = "/var/lib/clickhouse"
)

type Config struct {
	Host, Username, Password string
	Port                     int
}

type Client struct {
	Config     *Config
	Connection *sqlx.DB
	uid        *int
	gid        *int
}

type Table struct {
	UUID         string   `db:"uuid"`
	Database     string   `db:"database"`
	Name         string   `db:"name"`
	Query        string   `db:"create_table_query"`
	MetadataPath string   `db:"metadata_path"`
	DataPaths    []string `db:"data_paths"`
}

func New(clickhouseConf *Config) *Client {
	return &Client{
		Config: clickhouseConf,
	}
}

func (clickhouse *Client) Connect(database string) error {
	connectionUrl := fmt.Sprintf(
		"tcp://%s:%d?username=%s&password=%s&database=%s&connection_open_strategy=in_order",
		clickhouse.Config.Host,
		clickhouse.Config.Port,
		clickhouse.Config.Username,
		clickhouse.Config.Password,
		database,
	)
	connection, err := sqlx.Open("clickhouse", connectionUrl)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	if err = connection.Ping(); err != nil {
		if exception, ok := err.(*clickhouseGo.Exception); ok {
			log.Errorf("[%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		} else {
			log.Errorf("%+v", err)
		}
		return err
	}
	clickhouse.Connection = connection
	return nil
}

func (clickhouse *Client) CloseConnection() {
	if err := clickhouse.Connection.Close(); err != nil {
		log.Errorf("%+v", err)
	}
}

func (clickhouse *Client) Freeze(database string, tables []Table) error {
	fmt.Print("Freeze tables...")
	for _, table := range tables {
		query := fmt.Sprintf("ALTER TABLE `%s`.`%s` FREEZE", database, table.Name)
		if _, err := clickhouse.Connection.Exec(query); err != nil {
			helper.ColoredPrintln(helper.ColorRed, "error!")
			log.Errorf("can't freeze partition on '%s.%s': %v", database, table.Name, err)
			return err
		}
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (clickhouse *Client) GetTables(database string) (tables []Table, err error) {
	fmt.Print("Get tables...")
	query := fmt.Sprintf("SELECT uuid, database, name, metadata_path, data_paths, replaceRegexpOne(create_table_query, 'CREATE TABLE (\\\\w+).(\\\\w+) \\(', 'CREATE TABLE IF NOT EXISTS \\\\2 \\(') as create_table_query FROM system.tables WHERE is_temporary=0 AND database='%s'", database)
	if err := clickhouse.Connection.Select(&tables, query); err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		log.Errorf("can't get tables for database '%s': %v", database, err)
		return nil, err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return tables, nil
}

func (clickhouse *Client) DropAllData(database, cluster string, inCluster bool) error {
	var onCluster string
	fmt.Print("Drop all tables\t\t...")
	if inCluster {
		onCluster = fmt.Sprintf(" ON CLUSTER %s", cluster)
	}
	dropDatabaseQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s %s SYNC", database, onCluster)
	if _, err := clickhouse.Connection.Exec(dropDatabaseQuery); err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		log.Errorf("can't drop database '%s': %v", database, err)
		return err
	}
	createDatabaseQuery := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s %s", database, onCluster)
	if _, err := clickhouse.Connection.Exec(createDatabaseQuery); err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		log.Errorf("can't create database '%s': %v", database, err)
		return err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (clickhouse *Client) RestoreTablesSchemas(database, metadataPath string, cluster string, inCluster bool) error {
	var onCluster string
	fmt.Print("Restore tables schemas\t...")
	if inCluster {
		onCluster = fmt.Sprintf("ON CLUSTER %s", cluster)
	}
	if err := filepath.Walk(metadataPath, func(filePath string, fileInfo os.FileInfo, err error) error {
		fileExt := filepath.Ext(filePath)
		if fileExt != ".sql" {
			return nil
		}
		query, err := helper.ReadFile(filePath)
		if err != nil {
			log.Errorf("can't create table schema from file '%s': %v", filePath, err)
			return err
		}
		re := regexp.MustCompile(`(?m)CREATE TABLE IF NOT EXISTS (\w+) \(`)
		substitution := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.$1 %s (", database, onCluster)
		query = re.ReplaceAllString(query, substitution)
		re = regexp.MustCompile(`(?m)/clickhouse/tables/[a-z0-9-]+/{shard}`)
		substitution = "/clickhouse/tables/{uuid}/{shard}"
		query = re.ReplaceAllString(query, substitution)
		if _, err := clickhouse.Connection.Exec(query); err != nil {
			log.Errorf("can't create table schema from file '%s': %v", filePath, err)
			return err
		}
		return nil
	}); err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (clickhouse *Client) RestoreTablesData(database, dataPath string) error {
	fmt.Print("Restore tables data\t...")
	databasePath := path.Join(DefaultDataPath, "data", database)
	var metaPath = strings.Replace(dataPath, "data", "metadata", 1)
	var tableIdsPath = strings.Replace(dataPath, "data", "tables", 1)

	metaFiles, err := ioutil.ReadDir(metaPath)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}

	for _, metaFile := range metaFiles {
		tableName := strings.TrimSuffix(metaFile.Name(), filepath.Ext(metaFile.Name()))
		metaTablePath := path.Join(tableIdsPath, strings.Join([]string{tableName, "uuid"}, "."))
		tableUuid, err := helper.ReadFile(metaTablePath)
		if err != nil {
			log.Errorf("%+v", err)
			return err
		}

		tableDirName := string([]rune(tableUuid)[:3])
		srcTablePath := path.Join(dataPath, tableDirName, tableUuid)
		dstTablePath := path.Join(databasePath, tableName, "detached")

		if err := filepath.Walk(srcTablePath, func(filePath string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				log.Errorf("%+v", err)
				return err
			}
			if filePath == srcTablePath {
				return nil
			}
			relativePath := strings.Trim(strings.TrimPrefix(filePath, srcTablePath), "/")
			dstPath := path.Join(dstTablePath, relativePath)
			if fileInfo.IsDir() {
				if err = os.MkdirAll(dstPath, 0750); err != nil {
					log.Errorf("%+v", err)
					return err
				}
				if err = clickhouse.Chown(dstPath); err != nil {
					return err
				}
				return nil
			}
			if !fileInfo.Mode().IsRegular() {
				return nil
			}
			if err = os.Rename(filePath, dstPath); err != nil {
				log.Errorf("%+v", err)
				return err
			}
			if err = clickhouse.Chown(dstPath); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		backupPath := strings.Replace(dataPath, "data", "", 1)
		if err = os.RemoveAll(backupPath); err != nil {
			log.Errorf("%+v", err)
			return err
		}
		partitionsDirs, err := ioutil.ReadDir(dstTablePath)
		if err != nil {
			log.Errorf("%+v", err)
			return err
		}
		for _, partitionDir := range partitionsDirs {
			re := regexp.MustCompile(`(?m)(\d+)([_\d]+)`)
			partition := re.ReplaceAllString(partitionDir.Name(), "$1")
			query := fmt.Sprintf("ALTER TABLE %s.%s ATTACH PARTITION %s", database, tableName, partition)
			if _, err := clickhouse.Connection.Exec(query); err != nil {
				log.Errorf("%+v", err)
				return err
			}
		}
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (clickhouse *Client) GetClusters() (clusters []string, err error) {
	if err := clickhouse.Connection.Select(&clusters, "SELECT DISTINCT cluster FROM system.clusters"); err != nil {
		log.Errorf("can't get clusters: %+v", err)
		return nil, err
	}
	return clusters, nil
}

func (clickhouse *Client) Chown(filename string) error {
	if clickhouse.uid == nil || clickhouse.gid == nil {
		info, err := os.Stat(path.Join(DefaultDataPath, "data"))
		if err != nil {
			log.Errorf("%+v", err)
			return err
		}
		stat := info.Sys().(*syscall.Stat_t)
		uid := int(stat.Uid)
		gid := int(stat.Gid)
		clickhouse.uid = &uid
		clickhouse.gid = &gid
	}
	if err := os.Chown(filename, *clickhouse.uid, *clickhouse.gid); err != nil {
		log.Errorf("%+v", err)
		return err
	}
	return nil
}

func (clickhouse *Client) GetDatabases() (databases []string, err error) {
	if err := clickhouse.Connection.Select(&databases, "SELECT name FROM system.databases WHERE name NOT IN ('_temporary_and_external_tables', 'system')"); err != nil {
		log.Errorf("can't get databases: %+v", err)
		return nil, err
	}
	return databases, nil
}
