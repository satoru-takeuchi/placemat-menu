//go:generate statik -f -src=./public

package main

import (
	"bufio"
	"encoding/json"
	"errors"
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
	"/static/Makefile",
	"/static/bashrc",
	"/static/rkt-fetch",
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
	clusterFile, err := os.Create(filepath.Join(*flagOutDir, "cluster.yml"))
	if err != nil {
		return err
	}
	defer clusterFile.Close()
	err = menu.ExportCluster(clusterFile, ta)
	if err != nil {
		return err
	}
	err = export(statikFS, "/templates/bird_vm.conf", "bird_vm.conf", ta)
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

	networkConfigFile, err := os.Create(filepath.Join(*flagOutDir, "network.yml"))
	if err != nil {
		return err
	}
	defer networkConfigFile.Close()
	err = menu.ExportNetworkConfig(networkConfigFile)
	if err != nil {
		return err
	}

	err = func() error {
		seedFile, err := os.Create(filepath.Join(*flagOutDir, "seed_operation.yml"))
		if err != nil {
			return err
		}
		defer seedFile.Close()
		return menu.ExportOperationSeed(seedFile, ta)
	}()
	if err != nil {
		return err
	}

	for rackIdx, rack := range ta.Racks {
		err := func() error {
			seedFile, err := os.Create(filepath.Join(*flagOutDir, fmt.Sprintf("seed_%s-boot.yml", rack.Name)))
			if err != nil {
				return err
			}
			defer seedFile.Close()
			return menu.ExportBootSeed(seedFile, &ta.Account, &rack)
		}()
		if err != nil {
			return err
		}

		err = exportJSON(
			"ext-vm.ign",
			menu.ExternalNodeIgnition(ta.Account, ta.Network.Endpoints.External))
		if err != nil {
			return err
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

		err = export(statikFS, "/templates/bird_rack-node.conf",
			fmt.Sprintf("bird_rack%d-node.conf", rackIdx),
			menu.BIRDRackTemplateArgs{Args: *ta, RackIdx: rackIdx})
		if err != nil {
			return err
		}
		for csIdx, cs := range rack.CSList {
			err = exportJSON(
				fmt.Sprintf("rack%d-cs%d.ign", rackIdx, csIdx+1),
				menu.CSNodeIgnition(ta.Account, rack, cs))
			if err != nil {
				return err
			}
		}
		for ssIdx, ss := range rack.SSList {
			err = exportJSON(
				fmt.Sprintf("rack%d-ss%d.ign", rackIdx, ssIdx+1),
				menu.SSNodeIgnition(ta.Account, rack, ss))
			if err != nil {
				return err
			}
		}
	}
	return copyStatics(statikFS, staticFiles, *flagOutDir)
}

func exportJSON(output string, ignition menu.Ignition) error {
	f, err := os.Create(filepath.Join(*flagOutDir, output))
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(ignition)
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
