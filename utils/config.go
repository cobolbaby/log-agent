package utils

import (
	"github.com/go-ini/ini"
	"log"
	"os"
	"path/filepath"
)

var (
	iniCfg *ini.File
)

func ConfigMgr() *ini.File {
	if iniCfg != nil {
		return iniCfg
	}
	filename := filepath.Join("conf", "logagent.ini")
	if len(os.Args) == 3 && os.Args[1] == "-c" {
		filename = os.Args[2]
	}
	var err error
	iniCfg, err = ini.LoadSources(ini.LoadOptions{
		SkipUnrecognizableLines: true,
	}, filename)
	if err != nil {
		log.Fatalf("Failed to Load configuration: %v", err)
	}
	return iniCfg
}
