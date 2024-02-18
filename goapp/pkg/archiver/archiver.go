package archiver

import (
	"fmt"
	archiverLibrary "github.com/mholt/archiver/v3"
	log "github.com/sirupsen/logrus"
	"os"
)

const (
	CompressionFormatTAR   = "tar"
	CompressionFormatLZ4   = "lz4"
	CompressionFormatBZIP2 = "bzip2"
	CompressionFormatGZIP  = "gzip"
	CompressionFormatSZ    = "sz"
	CompressionFormatXZ    = "xz"
)

type Config struct {
	CompressionFormat string
	CompressionLevel  int
}

type Archiver struct {
	Config *Config
}

type File struct {
	Path, Name string
	Info       os.FileInfo
}

type Stringed interface {
	archiverLibrary.Writer
	String() string
}

func New(conf *Config) *Archiver {
	return &Archiver{
		Config: conf,
	}
}

func (archiver *Archiver) GetWriter() (archiverLibrary.Writer, error) {
	switch archiver.Config.CompressionFormat {
	case CompressionFormatTAR:
		return &archiverLibrary.Tar{
			MkdirAll: true,
		}, nil
	case CompressionFormatLZ4:
		return &archiverLibrary.TarLz4{
			CompressionLevel: archiver.Config.CompressionLevel,
			Tar:              archiverLibrary.NewTar(),
		}, nil
	case CompressionFormatBZIP2:
		return &archiverLibrary.TarBz2{
			CompressionLevel: archiver.Config.CompressionLevel,
			Tar:              archiverLibrary.NewTar(),
		}, nil
	case CompressionFormatGZIP:
		return &archiverLibrary.TarGz{
			CompressionLevel: archiver.Config.CompressionLevel,
			Tar:              archiverLibrary.NewTar(),
		}, nil
	case CompressionFormatSZ:
		return &archiverLibrary.TarSz{
			Tar: archiverLibrary.NewTar(),
		}, nil
	case CompressionFormatXZ:
		return &archiverLibrary.TarXz{
			Tar: archiverLibrary.NewTar(),
		}, nil
	}
	err := fmt.Errorf(
		"wrong compression_format, supported: '%s', '%s', '%s', '%s', '%s', '%s'",
		CompressionFormatTAR,
		CompressionFormatLZ4,
		CompressionFormatBZIP2,
		CompressionFormatGZIP,
		CompressionFormatSZ,
		CompressionFormatXZ,
	)
	log.Errorf("%+v", err)
	return nil, err
}

func (archiver *Archiver) GetExtension() string {
	var writer Stringed
	switch archiver.Config.CompressionFormat {
	case CompressionFormatLZ4:
		writer = &archiverLibrary.TarLz4{}
	case CompressionFormatBZIP2:
		writer = &archiverLibrary.TarBz2{}
	case CompressionFormatGZIP:
		writer = &archiverLibrary.TarGz{}
	case CompressionFormatSZ:
		writer = &archiverLibrary.TarSz{}
	case CompressionFormatXZ:
		writer = &archiverLibrary.TarXz{}
	default:
		writer = &archiverLibrary.Tar{}
	}

	return writer.String()
}

func (archiver *Archiver) Create(dstPath string) (archiverLibrary.Writer, error) {
	archive, err := os.Create(dstPath)
	if err != nil {
		log.Errorf("%+v", err)
		return nil, err
	}
	writer, err := archiver.GetWriter()
	if err != nil {
		return nil, err
	}
	if err := writer.Create(archive); err != nil {
		log.Errorf("%+v", err)
		return nil, err
	}
	return writer, nil
}

func (archiver *Archiver) AddFile(writer archiverLibrary.Writer, addingFile *File) error {
	file, err := os.Open(addingFile.Path)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	if addingFile.Info == nil {
		addingFile.Info, err = os.Stat(addingFile.Path)
		if err != nil {
			log.Errorf("%+v", err)
			return err
		}
	}
	if err := writer.Write(archiverLibrary.File{
		FileInfo: archiverLibrary.FileInfo{
			FileInfo:   addingFile.Info,
			CustomName: addingFile.Name,
		},
		ReadCloser: file,
	}); err != nil {
		log.Errorf("%+v", err)
		return err
	}
	return nil
}

func (archiver *Archiver) Unarchive(srcPath, dstPath string) error {
	if err := os.RemoveAll(dstPath); err != nil {
		log.Errorf("%+v", err)
		return err
	}
	if err := archiverLibrary.Unarchive(srcPath, dstPath); err != nil {
		log.Errorf("%+v", err)
		return err
	}
	return nil
}
