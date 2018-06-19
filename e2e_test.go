package menu

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
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
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(content1), string(content2), false)
		t.Error("unexpected file content: " + dmp.DiffPrettyText(diffs))
	}
}

func TestE2E(t *testing.T) {
	dir, err := ioutil.TempDir("", "placemat-menu-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	targets := []string{
		"cluster.yml",
		"bird_core.conf",
		"bird_spine1.conf",
		"bird_spine2.conf",
		"bird_rack0-tor1.conf",
		"bird_rack0-tor2.conf",
		"bird_rack1-tor1.conf",
		"bird_rack1-tor2.conf",
		"seed_rack0-boot.yml",
		"seed_rack1-boot.yml",
		"sabakan/ipam.json",
		"sabakan/dhcp.json",
		"sabakan/machines.json",
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
}
