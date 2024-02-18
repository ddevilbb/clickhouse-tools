package rsync

import (
	"bytes"
	"clickhouse-tools/internal/helper"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"path"
)

const (
	Name                      = "rsync"
	execCommandTypeUpload     = "upload"
	execCommandTypeDownload   = "download"
	execCommandTypeListString = "list_string"
)

type Options struct {
	Archive  bool   `cli_name:"--archive"`
	Verbose  bool   `cli_name:"--verbose"`
	Progress bool   `cli_name:"--progress"`
	ListOnly bool   `cli_name:"--list-only"`
	Rsh      string `cli_name:"--rsh"`
}

type Storage struct {
	config  *Config
	options *Options
}

type Config struct {
	Host, Username, Password, RemotePath, SSHKeyPath string
	UseSSH                                           bool
}
type ExecCommand struct {
	Type, Source, Destination string
}

func getArguments(options *Options) []string {
	var arguments []string
	if options.Archive {
		arguments = append(arguments, helper.GetAssociatedPropertyName(options, "Archive", "cli_name"))
	}
	if options.Verbose {
		arguments = append(arguments, helper.GetAssociatedPropertyName(options, "Verbose", "cli_name"))
	}
	if options.Progress {
		arguments = append(arguments, helper.GetAssociatedPropertyName(options, "Progress", "cli_name"))
	}
	if options.ListOnly {
		arguments = append(arguments, helper.GetAssociatedPropertyName(options, "ListOnly", "cli_name"))
	}
	if options.Rsh != "" {
		arguments = append(arguments, helper.GetAssociatedPropertyName(options, "Rsh", "cli_name"), options.Rsh)
	}
	return arguments
}

func New(conf *Config) *Storage {
	return &Storage{
		config: conf,
		options: &Options{
			Archive:  false,
			Verbose:  false,
			Progress: false,
			ListOnly: false,
			Rsh:      fmt.Sprintf("/usr/bin/ssh -i %s -p %d -o StrictHostKeyChecking=no -l %s", conf.SSHKeyPath, 22, "root"),
		},
	}
}

func (s *Storage) Upload(src string) error {
	fmt.Print("Upload backup by rsync...")
	s.options.Archive = true
	s.options.Verbose = true
	_, err := s.exec(&ExecCommand{
		Type:        execCommandTypeUpload,
		Source:      src,
		Destination: fmt.Sprintf("%s:%s", s.config.Host, s.config.RemotePath),
	})
	if err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (s *Storage) GetBackupListString() (string, error) {
	s.options.ListOnly = true
	std, err := s.exec(&ExecCommand{
		Type:   execCommandTypeListString,
		Source: fmt.Sprintf("%s:%s", s.config.Host, s.config.RemotePath),
	})
	if err != nil {
		return "", err
	}
	return std, nil
}

func (s *Storage) Download(destination, backupName string) error {
	fmt.Print("Download backup by rsync...")
	s.options.Archive = true
	s.options.Verbose = true
	_, err := s.exec(&ExecCommand{
		Type:        execCommandTypeDownload,
		Source:      fmt.Sprintf("%s:%s", s.config.Host, path.Join(s.config.RemotePath, backupName)),
		Destination: destination,
	})
	if err != nil {
		helper.ColoredPrintln(helper.ColorRed, "error!")
		return err
	}
	helper.ColoredPrintln(helper.ColorGreen, "done!")
	return nil
}

func (s *Storage) exec(execCommand *ExecCommand) (string, error) {
	var (
		stdout, stderr bytes.Buffer
		options        []string
	)
	switch execCommand.Type {
	case execCommandTypeUpload:
		fallthrough
	case execCommandTypeDownload:
		options = append(getArguments(s.options), execCommand.Source, execCommand.Destination)
		break
	case execCommandTypeListString:
		options = append(getArguments(s.options), execCommand.Source)
		break
	default:
		err := fmt.Errorf("unsupported exec command type '%s'", execCommand.Type)
		log.Errorf("%+v", err)
		return "", err
	}
	cmd := exec.Command("rsync", options...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%+v", err)
		return stderr.String(), err
	}
	return stdout.String(), nil
}
