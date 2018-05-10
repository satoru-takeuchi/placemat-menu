package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/log"
	menu "github.com/cybozu-go/placemat-menu"
)

const (
	staticFilesSource   = "statics"
	templateFilesSource = "templates"
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
	m, err := menu.ReadYAML(bufio.NewReader(f))
	if err != nil {
		log.ErrorExit(err)
	}

	ta, err := menu.ToTemplateArgs(m)
	if err != nil {
		log.ErrorExit(err)
	}

	err = export("Makefile", "Makefile", ta)
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

	err = export("ign.libsonnet", "ign.libsonnet", ta)
	if err != nil {
		log.ErrorExit(err)
	}

	err = export("bird_vm.conf", "bird_vm.conf", ta)
	if err != nil {
		log.ErrorExit(err)
	}

	err = export("ext-vm.jsonnet", "ext-vm.jsonnet", ta)
	if err != nil {
		log.ErrorExit(err)
	}

	for spineIdx := range ta.Spines {
		err = export("bird_spine.conf",
			fmt.Sprintf("bird_spine%d.conf", spineIdx+1),
			menu.BIRDSpineTemplateArgs{Args: *ta, SpineIdx: spineIdx})
		if err != nil {
			log.ErrorExit(err)
		}
	}

	for rackIdx, rack := range ta.Racks {
		err = export("rack-boot.jsonnet",
			fmt.Sprintf("rack%d-boot.jsonnet", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			log.ErrorExit(err)
		}

		err = export("bird_rack-tor1.conf",
			fmt.Sprintf("bird_rack%d-tor1.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			log.ErrorExit(err)
		}

		err = export("bird_rack-tor2.conf",
			fmt.Sprintf("bird_rack%d-tor2.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			log.ErrorExit(err)
		}

		err = export("bird_rack-node.conf",
			fmt.Sprintf("bird_rack%d-node.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			log.ErrorExit(err)
		}
		for csIdx, cs := range rack.CSList {
			err = export("rack-node.jsonnet",
				fmt.Sprintf("rack%d-cs%d.jsonnet", rackIdx, csIdx),
				menu.NodeTemplateArgs{rack, cs})
			if err != nil {
				log.ErrorExit(err)
			}
		}
		for ssIdx, ss := range rack.SSList {
			err = export("rack-node.jsonnet",
				fmt.Sprintf("rack%d-ss%d.jsonnet", rackIdx, ssIdx),
				menu.NodeTemplateArgs{rack, ss})
			if err != nil {
				log.ErrorExit(err)
			}
		}
	}

	err = copyStatics(*flagOutDir)
	if err != nil {
		log.ErrorExit(err)
	}
}

func export(inputFileName string, outputFileName string, args interface{}) error {
	f, err := os.Create(filepath.Join(*flagOutDir, outputFileName))
	if err != nil {
		return err
	}
	defer f.Close()
	t := template.Must(template.ParseFiles(filepath.Join(templateFilesSource, inputFileName)))
	return menu.Export(t, args, f)
}

func copyStatics(outputDirName string) error {
	fileInfos, err := ioutil.ReadDir(staticFilesSource)
	if err != nil {
		return err
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			return errors.New("cannot copy directory from " + staticFilesSource)
		}

		src, err := os.Open(filepath.Join(staticFilesSource, fileInfo.Name()))
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(filepath.Join(outputDirName, fileInfo.Name()))
		if err != nil {
			return err
		}
		defer dst.Close()

		err = dst.Chmod(fileInfo.Mode())
		if err != nil {
			return err
		}

		_, err = io.Copy(dst, src)
		if err != nil {
			return err
		}
	}

	return nil
}
