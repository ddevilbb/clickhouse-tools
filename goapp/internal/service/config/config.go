package config

import (
	"clickhouse-tools/internal/service/clickhouse"
	"clickhouse-tools/internal/service/storage/rsync"
	"clickhouse-tools/internal/service/storage/s3"
	"clickhouse-tools/pkg/archiver"
	"clickhouse-tools/pkg/elk_writer"
	"clickhouse-tools/pkg/encryptor"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Application struct {
	Clickhouse *clickhouse.Config
	Rsync      *rsync.Config
	Archiver   *archiver.Config
	S3         *s3.Config
	Encryption *encryptor.Config
	ElkWriter  *elk_writer.Config
}

func New() *Application {
	exec, err := os.Executable()
	if err != nil {
		log.Fatalln(err)
	}
	path := filepath.Dir(exec)
	if err := godotenv.Load(path + "/.env"); err != nil {
		log.Fatalf("No '%s/.env' file", path)
	}

	return &Application{
		Clickhouse: &clickhouse.Config{
			Host:     getEnvVarAsString("CLICKHOUSE_HOST", ""),
			Port:     getEnvVarAsInt("CLICKHOUSE_PORT", 0),
			Username: getEnvVarAsString("CLICKHOUSE_USERNAME", ""),
			Password: getEnvVarAsString("CLICKHOUSE_PASSWORD", ""),
		},
		Rsync: &rsync.Config{
			Host:       getEnvVarAsString("RSYNC_HOST", ""),
			Username:   getEnvVarAsString("RSYNC_USER", ""),
			Password:   getEnvVarAsString("RSYNC_PASSWORD", ""),
			RemotePath: getEnvVarAsString("RSYNC_REMOTE_PATH", ""),
			SSHKeyPath: getEnvVarAsString("RSYNC_SSH_KEY_PATH", ""),
			UseSSH:     getEnvVarAsBool("RSYNC_USE_SSH", false),
		},
		Archiver: &archiver.Config{
			CompressionFormat: getEnvVarAsString("ARCHIVER_COMPRESSION_FORMAT", "tar"),
			CompressionLevel:  getEnvVarAsInt("ARCHIVER_COMPRESSION_LEVEL", 9),
		},
		S3: &s3.Config{
			Write: &s3.Keys{
				AccessKey: getEnvVarAsString("S3_ACCESS_KEY_WRITE", ""),
				SecretKey: getEnvVarAsString("S3_SECRET_KEY_WRITE", ""),
			},
			Read: &s3.Keys{
				AccessKey: getEnvVarAsString("S3_ACCESS_KEY_READ", ""),
				SecretKey: getEnvVarAsString("S3_SECRET_KEY_READ", ""),
			},
			Endpoint:                getEnvVarAsString("S3_ENDPOINT", ""),
			Bucket:                  getEnvVarAsString("S3_BUCKET", ""),
			Directory:               getEnvVarAsString("S3_DIRECTORY", ""),
			Region:                  getEnvVarAsString("S3_REGION", ""),
			ACL:                     getEnvVarAsString("S3_ACL", "private"),
			DisableSSL:              getEnvVarAsBool("S3_DISABLE_SSL", false),
			DisableCertVerification: getEnvVarAsBool("S3_DISABLE_CERT_VERIFICATION", false),
			PartSize:                getEnvVarAsInt64("S3_PART_SIZE", 100*1024*1024),
			SSE:                     getEnvVarAsString("S3_SERVER_SIDE_ENCRYPTION", ""),
			ForcePathStyle:          getEnvVarAsBool("S3_FORCE_PATH_STYLE", true),
		},
		Encryption: &encryptor.Config{
			SecretKey:  getEnvVarAsString("ENCRYPTION_SECRET_KEY", ""),
			BufferSize: getEnvVarAsInt("ENCRYPTION_BUFFER_SIZE", 500*1024*1024),
		},
		ElkWriter: &elk_writer.Config{
			ConnectionNetwork: getEnvVarAsString("ELK_CONNECTION_NETWORK", ""),
			ConnectionUrl:     getEnvVarAsString("ELK_CONNECTION_URL", ""),
		},
	}
}

func getEnvVarAsString(name, defaultValue string) string {
	if value, exists := os.LookupEnv(name); exists {
		return value
	}
	return defaultValue
}

func getEnvVarAsInt(name string, defaultValue int) int {
	valueString := getEnvVarAsString(name, "")
	if value, err := strconv.Atoi(valueString); err == nil {
		return value
	}
	return defaultValue
}

func getEnvVarAsInt64(name string, defaultValue int64) int64 {
	valueString := getEnvVarAsString(name, "")
	if value, err := strconv.ParseInt(valueString, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvVarAsBool(name string, defaultValue bool) bool {
	valueString := getEnvVarAsString(name, "")
	if value, err := strconv.ParseBool(valueString); err == nil {
		return value
	}
	return defaultValue
}

func getEnvVarAsSlice(name string, defaultValue []string, sep string) []string {
	valueString := getEnvVarAsString(name, "")
	if valueString == "" {
		return defaultValue
	}
	value := strings.Split(valueString, sep)
	return value
}
