package main

import (
	"bufio"
	"flag"
	"os"
	"text/template"

	"github.com/cybozu-go/log"
	menu "github.com/cybozu-go/placemat-menu"
)

var (
	flagConfig = flag.String("f", "", "Template file for placemat-menu")
)

func main() {
	flag.Parse()
	f, err := os.Open(*flagConfig)
	if err != nil {
		log.ErrorExit(err)
	}
	defer f.Close()
	m, err := readYAML(bufio.NewReader(f))
	if err != nil {
		log.ErrorExit(err)
	}

	ta, err := menu.MenuToTemplateArgs(m)
	if err != nil {
		log.ErrorExit(err)
	}

	t := template.Must(template.ParseFiles("templates/cluster.yml"))
	err = menu.Export(t, ta)
	if err != nil {
		log.ErrorExit(err)
	}
}
