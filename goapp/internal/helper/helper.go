package helper

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
)

const (
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorGreen     = "\033[32m"
	ColorYellow    = "\033[33m"
	ColorBlue      = "\033[34m"
	ColorPurple    = "\033[35m"
	ColorCyan      = "\033[36m"
	ColorWhite     = "\033[37m"
	BufferSize     = 500000000
	UuidRegExpFile = `([\w\d\D\s]+ReplicatedMergeTree\('/clickhouse/tables/)([\w\d-]+)(/{shard}/[\w\d\D\s]+)`
)

func ColoredPrintln(color string, message string) {
	fmt.Printf("%s%s%s\n", color, message, ColorReset)
}

func ColoredPrint(color string, message string) {
	fmt.Printf("%s%s%s", color, message, ColorReset)
}

func CreateFile(filename string, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(file)
	if _, err = file.WriteString(content); err != nil {
		log.Errorf("%+v", err)
		return err
	}
	return nil
}

func CopyFile(srcFile string, dstFile string) error {
	sourceFileStat, err := os.Stat(srcFile)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		err := fmt.Errorf("%s is not a regular file", srcFile)
		log.Errorf("%+v", err)
		return err
	}
	source, err := os.Open(srcFile)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	defer func(source *os.File) {
		err := source.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(source)
	destination, err := os.Create(dstFile)
	if err != nil {
		log.Errorf("%+v", err)
		return err
	}
	defer func(destination *os.File) {
		err := destination.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(destination)

	buf := make([]byte, BufferSize)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			log.Errorf("%+v", err)
			return err
		}
		if n == 0 {
			break
		}
		if _, err := destination.Write(buf[:n]); err != nil {
			log.Errorf("%+v", err)
			return err
		}
	}
	return nil
}

func GetAssociatedPropertyName(object interface{}, propertyName, assocName string) string {
	reflectValue := reflect.Indirect(reflect.ValueOf(object))
	property, _ := reflectValue.Type().FieldByName(propertyName)
	return property.Tag.Get(assocName)
}

func FormatBytes(i int64) (result string) {
	const (
		KiB = 1024
		MiB = 1048576
		GiB = 1073741824
		TiB = 1099511627776
	)
	switch {
	case i >= TiB:
		result = fmt.Sprintf("%.02f TiB", float64(i)/TiB)
	case i >= GiB:
		result = fmt.Sprintf("%.02f GiB", float64(i)/GiB)
	case i >= MiB:
		result = fmt.Sprintf("%.02f MiB", float64(i)/MiB)
	case i >= KiB:
		result = fmt.Sprintf("%.02f KiB", float64(i)/KiB)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	return result
}

func GetUuidFromSqlFile(metaTablePath string) (string, error) {
	query, err := ReadFile(metaTablePath)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(UuidRegExpFile)
	uuid := re.ReplaceAllString(query, "$2")

	return uuid, nil
}

func ReadFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(file)
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}
