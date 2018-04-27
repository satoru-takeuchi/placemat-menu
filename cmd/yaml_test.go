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
}

func TestYAML(t *testing.T) {
	t.Run("network", testUnmarshalNetwork)
	// t.Run("inventory", testUnmarshalInventory)
	// t.Run("node", testUnmarshalNode)
	// t.Run("account", testUnmarshalAccount)
}
