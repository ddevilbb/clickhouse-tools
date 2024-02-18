package elk_writer

import (
	"net"
	"os"
)

type Config struct {
	ConnectionNetwork string
	ConnectionUrl     string
}

type ElkWriter struct {
	ConnectionNetwork string
	ConnectionUrl     string
}

func New(conf *Config) *ElkWriter {
	return &ElkWriter{
		ConnectionNetwork: conf.ConnectionNetwork,
		ConnectionUrl:     conf.ConnectionUrl,
	}
}

func (w *ElkWriter) Write(p []byte) (n int, err error) {
	conn, err := net.Dial(w.ConnectionNetwork, w.ConnectionUrl)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error())
		conn = nil
	}

	_, _ = os.Stderr.Write(p)

	if conn == nil {
		return n, err
	}

	defer conn.Close()

	n, err = conn.Write(p)
	if err != nil {
		if _, err := os.Stderr.WriteString(err.Error()); err != nil {
			return n, err
		}
	}

	return n, err
}
