package logger

import (
	"clickhouse-tools/internal/service/config"
	"clickhouse-tools/pkg/elk_writer"
	log "github.com/sirupsen/logrus"
)

type jsonLogger struct {
	Type      string
	formatter log.Formatter
}

func Init(conf *config.Application) {
	log.SetFormatter(jsonLogger{
		Type:      "clickhouse-backup",
		formatter: &log.JSONFormatter{},
	})
	log.SetOutput(elk_writer.New(conf.ElkWriter))
	log.SetReportCaller(true)
}

func (l jsonLogger) Format(entry *log.Entry) ([]byte, error) {
	entry.Data["type"] = l.Type

	return l.formatter.Format(entry)
}
