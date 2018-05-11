//go:generate statik -src=./public

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"net/http"

	menu "github.com/cybozu-go/placemat-menu"
	_ "github.com/cybozu-go/placemat-menu/cmd/placemat-menu/statik"
	"github.com/rakyll/statik/fs"
)

const (
	templateFilesSource = "templates"
)

var staticFiles = []string{
	"/static/bashrc",
	"/static/ign.libsonnet",
	"/static/rkt-fetch",
	"/static/setup-iptables",
	"/static/setup-rp-filter",
}

var (
	flagConfig = flag.String("f", "", "Template file for placemat-menu")
	flagOutDir = flag.String("o", ".", "Directory for output files")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	staticFS, err := fs.New()
	if err != nil {
		return err
	}

	fi, err := os.Stat(*flagOutDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return errors.New(*flagOutDir + "is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(*flagOutDir, 0755)
		if err != nil {
			return err
		}
	default:
		return err
	}
	f, err := os.Open(*flagConfig)
	if err != nil {
		return err
	}
	defer f.Close()
	m, err := menu.ReadYAML(bufio.NewReader(f))
	if err != nil {
		return err
	}
	ta, err := menu.ToTemplateArgs(m)
	if err != nil {
		return err
	}
	err = export("Makefile", "Makefile", ta)
	if err != nil {
		return err
	}
	err = export("cluster.yml", "cluster.yml", ta)
	if err != nil {
		return err
	}
	err = export("bird_vm.conf", "bird_vm.conf", ta)
	if err != nil {
		return err
	}
	err = export("ext-vm.jsonnet", "ext-vm.jsonnet", ta)
	if err != nil {
		return err
	}
	for spineIdx := range ta.Spines {
		err = export("bird_spine.conf",
			fmt.Sprintf("bird_spine%d.conf", spineIdx+1),
			menu.BIRDSpineTemplateArgs{Args: *ta, SpineIdx: spineIdx})
		if err != nil {
			return err
		}
	}
	for rackIdx, rack := range ta.Racks {
		err = export("rack-boot.jsonnet",
			fmt.Sprintf("rack%d-boot.jsonnet", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}

		err = export("bird_rack-tor1.conf",
			fmt.Sprintf("bird_rack%d-tor1.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}

		err = export("bird_rack-tor2.conf",
			fmt.Sprintf("bird_rack%d-tor2.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}

		err = export("bird_rack-node.conf",
			fmt.Sprintf("bird_rack%d-node.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}
		for csIdx, cs := range rack.CSList {
			err = export("rack-node.jsonnet",
				fmt.Sprintf("rack%d-cs%d.jsonnet", rackIdx, csIdx+1),
				menu.NodeTemplateArgs{rack, cs, ta.Account})
			if err != nil {
				return err
			}
		}
		for ssIdx, ss := range rack.SSList {
			err = export("rack-node.jsonnet",
				fmt.Sprintf("rack%d-ss%d.jsonnet", rackIdx, ssIdx+1),
				menu.NodeTemplateArgs{rack, ss, ta.Account})
			if err != nil {
				return err
			}
		}
	}
	return copyStatics(staticFS, staticFiles, *flagOutDir)
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

func copyStatics(fs http.FileSystem, inputs []string, outputDirName string) error {
	for _, fileName := range inputs {
		err := copyStatic(fs, fileName, outputDirName)
		if err != nil {
			return err
		}

	}

	return nil
}

func copyStatic(fs http.FileSystem, fileName string, outputDirName string) error {
	src, err := fs.Open(fileName)
	if err != nil {
		return err
	}
	defer src.Close()
	fi, err := src.Stat()
	if err != nil {
		return err
	}

	dst, err := os.Create(filepath.Join(outputDirName, filepath.Base(fileName)))
	if err != nil {
		return err
	}
	defer dst.Close()

	err = dst.Chmod(fi.Mode())
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)
	return err
}
