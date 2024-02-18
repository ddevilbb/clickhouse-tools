package storage

import (
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/internal/service/storage/rsync"
	"clickhouse-tools/internal/service/storage/s3"
	"clickhouse-tools/pkg/encryptor"
	"fmt"
	log "github.com/sirupsen/logrus"
)

type Interface interface {
	Upload(src string) error
	GetBackupListString() (string, error)
	Download(destination, backupName string) error
}

func InitStorage(conf *config.Application, storageName string) (Interface, error) {
	switch storageName {
	case rsync.Name:
		return rsync.New(conf.Rsync), nil
	case s3.Name:
		enc := encryptor.New(conf.Encryption)
		return s3.New(conf.S3, enc), nil
	default:
		err := fmt.Errorf("unsupported storage name '%s'", storageName)
		log.Errorf("%+v", err)
		return nil, err
	}
}
