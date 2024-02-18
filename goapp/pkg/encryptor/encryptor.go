package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/scrypt"
	"io"
	"os"
	"strings"
)

type Config struct {
	SecretKey  string
	BufferSize int
}

type Encryptor struct {
	Config *Config
}

func New(config *Config) *Encryptor {
	return &Encryptor{
		Config: config,
	}
}

func (encryptor *Encryptor) EncryptFile(src string) (string, error) {
	encSrc := src + ".enc"
	srcFile, err := os.Open(src)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	defer func(srcFile *os.File) {
		err := srcFile.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(srcFile)

	key, salt, err := encryptor.deriveKey([]byte(encryptor.Config.SecretKey), nil)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		log.Errorf("%+v", err)
		return "", err
	}

	dstFile, err := os.OpenFile(encSrc, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	defer func(dstFile *os.File) {
		err := dstFile.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(dstFile)

	buf := make([]byte, encryptor.Config.BufferSize)
	stream := cipher.NewCTR(block, iv)
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			_, writeErr := dstFile.Write(buf[:n])
			if err != nil {
				log.Errorf("%+v", writeErr)
				return "", writeErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("%+v", err)
			return "", err
		}
	}
	if _, err = dstFile.Write(iv); err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	if _, err = dstFile.Write(salt); err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	return encSrc, nil
}

func (encryptor *Encryptor) DecryptFile(encSrc string) (string, error) {
	dstSrc := strings.TrimRight(encSrc, ".enc")
	encFile, err := os.Open(encSrc)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	defer func(encFile *os.File) {
		err := encFile.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(encFile)

	encFileStat, err := encFile.Stat()
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}

	salt := make([]byte, 32)
	saltStart := encFileStat.Size() - int64(len(salt))
	if _, err = encFile.ReadAt(salt, saltStart); err != nil {
		log.Errorf("%+v", err)
		return "", err
	}

	key, _, err := encryptor.deriveKey([]byte(encryptor.Config.SecretKey), salt)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}

	iv := make([]byte, block.BlockSize())
	msgLen := encFileStat.Size() - int64(len(iv)) - int64(len(salt))
	if _, err = encFile.ReadAt(iv, msgLen); err != nil {
		log.Errorf("%+v", err)
		return "", err
	}

	dstFile, err := os.OpenFile(dstSrc, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Errorf("%+v", err)
		return "", err
	}
	defer func(dstFile *os.File) {
		err := dstFile.Close()
		if err != nil {
			log.Errorf("%+v", err)
		}
	}(dstFile)

	buf := make([]byte, encryptor.Config.BufferSize)
	stream := cipher.NewCTR(block, iv)
	for {
		n, err := encFile.Read(buf)
		if n > 0 {
			if n > int(msgLen) {
				n = int(msgLen)
			}
			msgLen -= int64(n)
			stream.XORKeyStream(buf, buf[:n])
			if _, err := dstFile.Write(buf[:n]); err != nil {
				log.Errorf("%+v", err)
				return "", err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("%+v", err)
			return "", err
		}
	}
	return dstSrc, nil
}

func (encryptor *Encryptor) deriveKey(keyStr, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			log.Errorf("%+v", err)
			return nil, nil, err
		}
	}

	key, err := scrypt.Key(keyStr, salt, 1048576, 8, 1, 32)
	if err != nil {
		log.Errorf("%+v", err)
		return nil, nil, err
	}
	return key, salt, nil
}
