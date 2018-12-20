package compress

import (
	"bytes"
	"compress/gzip"
	"errors"
	"regexp"
)

var (
	NotCompressType = regexp.MustCompile(`(jpg|jpeg|gif|png|wmv|flv|zip|gz)$`)
)

const (
	GZIP_MIN_LENGTH = 1024
)

func CheckIfCompressExt(ext string) bool {
	return !NotCompressType.MatchString(ext)
}

func CheckIfCompressSize(leng int64) bool {
	return leng > GZIP_MIN_LENGTH
}

func GzipContent(originBuff []byte) ([]byte, error) {
	if len(originBuff) == 0 {
		return nil, errors.New("Compressed originBuff is null")
	}

	buf := new(bytes.Buffer)
	gw, err := gzip.NewWriterLevel(buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := gw.Write(originBuff); err != nil {
		gw.Close()
		return nil, err
	}
	// fix: Unexpected end of ZLIB input stream
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
