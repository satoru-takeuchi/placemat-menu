package main

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/log"
	menu "github.com/cybozu-go/placemat-menu"
)

var (
	flagConfig = flag.String("f", "", "Template file for placemat-menu")
	flagOutDir = flag.String("o", ".", "Directory for output files")
)

func main() {
	flag.Parse()

	fi, err := os.Stat(*flagOutDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			log.ErrorExit(errors.New(*flagOutDir + "is not a directory"))
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(*flagOutDir, 0755)
		if err != nil {
			log.ErrorExit(err)
		}
	default:
		log.ErrorExit(err)
	}

	f, err := os.Open(*flagConfig)
	if err != nil {
		log.ErrorExit(err)
	}
	defer f.Close()
	m, err := readYAML(bufio.NewReader(f))
	if err != nil {
		log.ErrorExit(err)
	}

	ta, err := menu.ToTemplateArgs(m)
	if err != nil {
		log.ErrorExit(err)
	}

	err = export("cluster.yml", "cluster.yml", ta)
	if err != nil {
		log.ErrorExit(err)
	}

	err = export("ign.jsonnet", "ign.jsonnet", ta)
	if err != nil {
		log.ErrorExit(err)
	}

}

func export(inputFileName string, outputFileName string, ta *menu.TemplateArgs) error {
	f, err := os.Create(filepath.Join(*flagOutDir, outputFileName))
	if err != nil {
		return err
	}
	defer f.Close()
	t := template.Must(template.ParseFiles(filepath.Join("templates", inputFileName)))
	return menu.Export(t, ta, f)
}
