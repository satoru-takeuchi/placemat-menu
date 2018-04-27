package main

import (
	"bufio"
	"os"
	"text/template"

	"github.com/cybozu-go/log"
)

func main() {
	const node = `
	{{range $index, $node:= .}}
	---
	kind: Node
	name: rack{{$index}}-{{$node.Type}}
	spec:
	 interfaces:
	   - rack0-node1
	   - rack0-node2
	 volumes:
	   - kind: image
	     name: root
	     spec:
	       image: coreos-image
	       copy-on-write: true
	   - kind: vvfat
	     name: common
	     spec:
	       folder: common-data
	   - kind: vvfat
	     name: local
	     spec:
	       folder: rack0-bird-data
	 ignition: rack0-boot.ign
	 resources:
	   cpu: {{$node.CPU}}
	   memory: {{$node.Memory}}
	{{end}}
	`

	f, err := os.Open("node.example.yml")
	if err != nil {
		log.ErrorExit(err)
	}
	defer f.Close()
	nodes, err := readYAML(bufio.NewReader(f))
	if err != nil {
		log.ErrorExit(err)
	}

	t := template.Must(template.New("test").Parse(node))
	t.Execute(os.Stdout, nodes)
}
