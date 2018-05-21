package menu

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	ignition "github.com/coreos/ignition/config/v2_2"
)

func assertFileEqual(t *testing.T, f1, f2 string) {
	content1, err := ioutil.ReadFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	content2, err := ioutil.ReadFile(f2)
	if err != nil {
		t.Fatal(err)
	}
	if string(content1) != string(content2) {
		t.Error("unexpected file content: " + filepath.Base(f1))
	}
}

func assertJSONFileEqual(t *testing.T, name1, name2 string) {
	var ign1, ign2 Ignition

	f1, err := os.Open(name1)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()
	f2, err := os.Open(name2)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	err = json.NewDecoder(f1).Decode(&ign1)
	if err != nil {
		t.Fatal(err)
	}
	err = json.NewDecoder(f2).Decode(&ign2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(ign1, ign2) {
		t.Error("unexpected file content: " + filepath.Base(f1.Name()))
	}
}

func assertValidIgnition(t *testing.T, path string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	_, rpt, err := ignition.Parse(content)
	if err != nil {
		t.Errorf("invalid ignition: %s: %s", filepath.Base(path), err)
	}

	if rpt.IsFatal() {
		t.Errorf("invalid ignition: %s: %s", filepath.Base(path), rpt.String())
	}

}

func TestE2E(t *testing.T) {
	dir, err := ioutil.TempDir("", "placemat-menu-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	targets := []string{
		"bird_rack0-node.conf",
		"bird_rack0-tor2.conf",
		"bird_rack1-tor1.conf",
		"bird_spine1.conf",
		"bird_vm.conf",
		"bird_rack0-tor1.conf",
		"bird_rack1-node.conf",
		"bird_rack1-tor2.conf",
		"bird_spine2.conf",
		"seed_rack0-boot.yml",
		"seed_rack1-boot.yml",
		"cluster.yml",
		"network.yml",
	}

	targetJSONs := []string{
		"ext-vm.ign",
		"rack0-cs1.ign",
		"rack0-cs2.ign",
		"rack1-cs1.ign",
		"rack1-cs2.ign",
		"rack1-ss1.ign",
		"rack1-ss2.ign",
	}

	cmd := exec.Command("go", "run", "cmd/placemat-menu/main.go", "-f", "example.yml", "-o", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range targets {
		f1 := filepath.Join(dir, f)
		f2 := filepath.Join("testdata", f)
		assertFileEqual(t, f1, f2)
	}

	for _, f := range targetJSONs {
		f1 := filepath.Join(dir, f)
		f2 := filepath.Join("testdata", f)
		assertJSONFileEqual(t, f1, f2)
	}
	for _, f := range targetJSONs {
		p := filepath.Join("testdata", f)
		assertValidIgnition(t, p)
	}
}
