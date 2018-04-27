package main

import (
	"net"
	"reflect"
	"testing"

	menu "github.com/cybozu-go/placemat-menu"
)

func testUnmarshalNetwork(t *testing.T) {
	t.Parallel()

	_, external, _ := net.ParseCIDR("10.0.0.0/24")
	_, node, _ := net.ParseCIDR("10.69.0.0/26")
	_, bastion, _ := net.ParseCIDR("10.72.48.0/26")
	_, loadbalancer, _ := net.ParseCIDR("10.72.32.0/20")
	_, ingress, _ := net.ParseCIDR("10.72.48.64/26")

	cases := []struct {
		source   string
		expected menu.NetworkMenu
	}{
		{
			source: `
kind: Network
spec:
  asn-base: 64600
  external: 10.0.0.0/24
  spine-tor: 10.0.1.0
  node: 10.69.0.0/26
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
`,
			expected: menu.NetworkMenu{
				ASNBase:      64600,
				External:     external,
				SpineTor:     net.ParseIP("10.0.1.0"),
				Node:         node,
				Bastion:      bastion,
				LoadBalancer: loadbalancer,
				Ingress:      ingress,
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNetwork([]byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# Invalid CIDR @ external
kind: Network
spec:
  asn-base: 64600
  external: 10.0.0.0
  spine-tor: 10.0.1.0
  node: 10.69.0.0/26
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
`,
		`
# Invalid network address @ node
kind: Network
spec:
  asn-base: 64600
  external: 10.0.0.0/24
  spine-tor: 10.0.1.0
  node: 10.69.0.1/26
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
`,
		`
# Invalid IP address @ spine-tor
kind: Network
spec:
  asn-base: 64600
  external: 10.0.0.0/24
  spine-tor: 10.0.1.0/31
  node: 10.69.0.0/26
  exposed:
    loadbalancer: 10.72.32.0/20
    bastion: 10.72.48.0/26
    ingress: 10.72.48.64/26
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalNetwork([]byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func testUnmarshalInventory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected menu.InventoryMenu
	}{
		{
			source: `
kind: Inventory
spec:
  spine: 3
  rack:
    - cs: 3
      ss: 0
    - cs: 2
      ss: 2
    - cs: 0
      ss: 3
`,
			expected: menu.InventoryMenu{
				Spine: 3,
				Rack: []menu.RackMenu{
					{CS: 3, SS: 0},
					{CS: 2, SS: 2},
					{CS: 0, SS: 3},
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalInventory([]byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# No spine
kind: Inventory
spec:
  spine: 0
  rack:
    - cs: 3
      ss: 0
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalInventory([]byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func testUnmarshalNode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected menu.NodeMenu
	}{
		{
			source: `
kind: Node
type: boot
spec:
  cpu: 1
  memory: 2G
`,
			expected: menu.NodeMenu{
				Type:   menu.BootNode,
				CPU:    1,
				Memory: "2G",
			},
		},
		{
			source: `
kind: Node
type: cs
spec:
  cpu: 2
  memory: 4G
`,
			expected: menu.NodeMenu{
				Type:   menu.CSNode,
				CPU:    2,
				Memory: "4G",
			},
		},
		{
			source: `
kind: Node
type: ss
spec:
  cpu: 1
  memory: 1G
`,
			expected: menu.NodeMenu{
				Type:   menu.SSNode,
				CPU:    1,
				Memory: "1G",
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNode([]byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# Invalid type
kind: Node
type: storage
spec:
  cpu: 2
  memory: 2G
`,
		`
# No CPU
kind: Node
type: cs
spec:
  cpu: 0
  memory: 2G
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalNode([]byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func testUnmarshalAccount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source   string
		expected menu.AccountMenu
	}{
		{
			source: `
kind: Account
spec:
  username: scott
  password: tiger
`,
			expected: menu.AccountMenu{
				UserName: "scott",
				Password: "tiger",
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalAccount([]byte(c.source))
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []string{
		`
# Empty username
kind: Account
spec:
  username:
  password: tiger
`,
	}

	for _, s := range errorSources {
		_, err := unmarshalAccount([]byte(s))
		if err == nil {
			t.Error("err == nil", s)
		}
	}
}

func TestYAML(t *testing.T) {
	t.Run("network", testUnmarshalNetwork)
	t.Run("inventory", testUnmarshalInventory)
	t.Run("node", testUnmarshalNode)
	t.Run("account", testUnmarshalAccount)
}
