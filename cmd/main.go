package main

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"text/template"

	"github.com/cybozu-go/log"
	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	"gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

type nodeConfig struct {
	Type     string `yaml:"type"`
	NodeSpec `yaml:"spec"`
}

type NodeSpec struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type Node struct {
	Type   string
	CPU    int
	Memory string
}

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

func readYAML(r *bufio.Reader) ([]Node, error) {
	c := baseConfig{}
	y := k8sYaml.NewYAMLReader(r)
	nodes := []Node{}
	for {
		data, err := y.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return nil, err
		}

		switch c.Kind {
		case "Node":
			node, err := unmarshalNode(data)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, *node)
		}
	}
	return nodes, nil
}
func unmarshalNode(data []byte) (*Node, error) {
	nodeConfig := nodeConfig{}
	err := yaml.Unmarshal(data, &nodeConfig)
	if err != nil {
		return nil, err
	}

	var node Node
	node.Type = nodeConfig.Type
	node.CPU, err = strconv.Atoi(nodeConfig.NodeSpec.CPU)
	if err != nil {
		return nil, err
	}
	node.Memory = nodeConfig.NodeSpec.Memory
	return &node, nil
}
