package s3

import (
	"bytes"
	"clickhouse-tools/internal/helper"
	"clickhouse-tools/pkg/encryptor"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

const (
	Name = "s3"
)

type Keys struct {
	AccessKey, SecretKey string
}

type Config struct {
	Write                                               *Keys
	Read                                                *Keys
	Bucket, Directory, Endpoint, Region, ACL, SSE       string
	ForcePathStyle, DisableSSL, DisableCertVerification bool
	PartSize                                            int64
}

type Storage struct {
	config    *Config
	encryptor *encryptor.Encryptor
}

func New(conf *Config, encryptor *encryptor.Encryptor) *Storage {
	return &Storage{
		config:    conf,
		encryptor: encryptor,
	}
}

func (s *Storage) Upload(src string) error {
	sess, err := s.connect(s.config.Write)
	if err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	encSrc, err := s.encryptor.EncryptFile(src)
	if err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	fmt.Print("Upload backup by s3...")
	file, err := os.Open(encSrc)
	if err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(file)
	fileStat, err := file.Stat()
	if err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	uploader := s3manager.NewUploader(sess)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(strings.Join([]string{s.config.Bucket, s.config.Directory}, "/") + "/"),
		ACL:    aws.String(s.config.ACL),
		Key:    aws.String(fileStat.Name()),
		Body:   file,
	})
	if err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	if err = os.Remove(encSrc); err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (s *Storage) GetBackupListString() (string, error) {
	var listBuffer bytes.Buffer
	sess, err := s.connect(s.config.Read)
	if err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return "", err
	}

	s3Client := s3.New(sess)

	list, err := s3Client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(s.config.Bucket),
		Prefix: aws.String(s.config.Directory),
	})

	if err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return "", err
	}

	for _, item := range list.Contents {
		date := item.LastModified.Format("2006/01/02 15:04:05")
		name := strings.ReplaceAll(*item.Key, strings.Join([]string{s.config.Directory, "/"}, ""), "")
		listBuffer.WriteString(date + " " + name + "\n\r")
	}
	return listBuffer.String(), nil
}

func (s *Storage) Download(destination, backupName string) error {
	sess, err := s.connect(s.config.Read)
	if err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}

	file, err := os.OpenFile(destination, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}

	fmt.Print("Download backup from s3...")
	downloader := s3manager.NewDownloader(sess, func(d *s3manager.Downloader) {
		d.PartSize = 128 * 1024 * 1024 // 128MB per part
		d.Concurrency = 10
	})
	if _, err := downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(strings.Join([]string{s.config.Bucket, s.config.Directory}, "/") + "/"),
		Key:    aws.String(backupName),
	}); err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}

	if err := file.Close(); err != nil {
		log.Errorf("%+v", err)
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}

	if _, err := s.encryptor.DecryptFile(destination); err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}

	if err := os.Remove(destination); err != nil {
		log.Errorf("%+v", err)
		return err
	}

	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (s *Storage) connect(keys *Keys) (*session.Session, error) {
	sess, err := session.NewSession(
		&aws.Config{
			Credentials: credentials.NewStaticCredentials(
				keys.AccessKey,
				keys.SecretKey,
				"",
			),
			Endpoint:         aws.String(s.config.Endpoint),
			Region:           aws.String(s.config.Region),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		})
	if err != nil {
		log.Errorf("%+v", err)
		return nil, err
	}
	return sess, nil
}
