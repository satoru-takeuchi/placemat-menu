package menu

import (
	"net"
	"testing"
)

func TestMakeHostAddressFromNetworkAddress(t *testing.T) {
	expected := "10.0.0.1/24"
	_, addr, _ := net.ParseCIDR("10.0.0.0/24")
	actual := makeHostAddressFromNetworkAddress(addr, 1)
	if expected != actual {
		t.Errorf("expected %v, actual %v", expected, actual)
	}
}

func TestMakeHostAddressFromIPAddress(t *testing.T) {
	expected := "10.0.0.1/24"
	ip := net.ParseIP("10.0.0.0")
	actual := makeHostAddressFromIPAddress(&ip, 1, 24)
	if expected != actual {
		t.Errorf("expected %v, actual %v", expected, actual)
	}
}
