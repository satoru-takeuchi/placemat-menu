//go:generate statik -f -src=./public

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/placemat-menu"
	_ "github.com/cybozu-go/placemat-menu/cmd/placemat-menu/statik"
	"github.com/rakyll/statik/fs"
)

var staticFiles = []string{
	"/static/setup-iptables",
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
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	opdir := filepath.Join(*flagOutDir, "operation")
	err = os.MkdirAll(opdir, 0755)
	if err != nil {
		return err
	}
	sabakanDir := filepath.Join(*flagOutDir, "sabakan")
	err = os.MkdirAll(sabakanDir, 0755)
	if err != nil {
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

	clusterFile, err := os.Create(filepath.Join(*flagOutDir, "cluster.yml"))
	if err != nil {
		return err
	}
	defer clusterFile.Close()
	err = menu.ExportCluster(clusterFile, ta)
	if err != nil {
		return err
	}

	err = export(statikFS, "/templates/setup-default-gateway", "setup-default-gateway-operation", ta.Core.OperationAddress)
	if err != nil {
		return err
	}
	err = export(statikFS, "/templates/setup-default-gateway", "setup-default-gateway-external", ta.Core.ExternalAddress)
	if err != nil {
		return err
	}

	err = export(statikFS, "/templates/bird_core.conf", "bird_core.conf", ta)
	if err != nil {
		return err
	}
	for spineIdx := range ta.Spines {
		err = export(statikFS, "/templates/bird_spine.conf",
			fmt.Sprintf("bird_spine%d.conf", spineIdx+1),
			menu.BIRDSpineTemplateArgs{Args: *ta, SpineIdx: spineIdx})
		if err != nil {
			return err
		}
	}

	networkFile, err := os.Create(filepath.Join(*flagOutDir, "network.yml"))
	if err != nil {
		return err
	}
	defer networkFile.Close()
	err = menu.ExportEmptyNetworkConfig(networkFile)
	if err != nil {
		return err
	}

	for rackIdx, rack := range ta.Racks {
		if ta.Boot.CloudInitTemplate != "" {
			arg := struct {
				Name string
				Rack menu.Rack
			}{
				fmt.Sprintf("%s-boot", rack.Name),
				rack,
			}
			err := exportFile(ta.Boot.CloudInitTemplate, fmt.Sprintf("seed_%s-boot.yml", rack.Name), arg)
			if err != nil {
				return err
			}
		}

		if ta.CS.CloudInitTemplate != "" {
			for _, cs := range rack.CSList {
				arg := struct {
					Name string
					Rack menu.Rack
				}{
					fmt.Sprintf("%s-%s", rack.Name, cs.Name),
					rack,
				}
				err := exportFile(ta.CS.CloudInitTemplate, fmt.Sprintf("seed_%s-%s.yml", rack.Name, cs.Name), arg)
				if err != nil {
					return err
				}
			}
		}

		if ta.SS.CloudInitTemplate != "" {
			for _, ss := range rack.SSList {
				arg := struct {
					Name string
					Rack menu.Rack
				}{
					fmt.Sprintf("%s-%s", rack.Name, ss.Name),
					rack,
				}
				err := exportFile(ta.SS.CloudInitTemplate, fmt.Sprintf("seed_%s-%s.yml", rack.Name, ss.Name), arg)
				if err != nil {
					return err
				}
			}
		}

		err = export(statikFS, "/templates/bird_rack-tor1.conf",
			fmt.Sprintf("bird_rack%d-tor1.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}

		err = export(statikFS, "/templates/bird_rack-tor2.conf",
			fmt.Sprintf("bird_rack%d-tor2.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}
	}

	err = menu.ExportSabakanData(sabakanDir, m, ta)
	if err != nil {
		return err
	}

	return copyStatics(statikFS, staticFiles, *flagOutDir)
}

func exportFile(input string, output string, args interface{}) error {
	f, err := os.Create(filepath.Join(*flagOutDir, output))
	if err != nil {
		return err
	}
	defer f.Close()

	templateFile, err := os.Open(input)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(templateFile)
	if err != nil {
		return err
	}

	tmpl, err := template.New(input).Parse(string(content))
	if err != nil {
		panic(err)
	}
	return tmpl.Execute(f, args)
}

func export(fs http.FileSystem, input string, output string, args interface{}) error {
	f, err := os.Create(filepath.Join(*flagOutDir, output))
	if err != nil {
		return err
	}
	defer f.Close()

	templateFile, err := fs.Open(input)
	if err != nil {
		return err
	}
	fi, err := templateFile.Stat()
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(templateFile)
	if err != nil {
		return err
	}

	tmpl, err := template.New(input).Parse(string(content))
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(f, args)
	if err != nil {
		return err
	}

	return f.Chmod(fi.Mode())
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
