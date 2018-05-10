package menu

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

func TestE2E(t *testing.T) {
	dir, err := ioutil.TempDir("", "placemat-menu-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	targets, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", "cmd/placemat-menu/main.go", "-f", "example.yml", "-o", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	cmd = exec.Command("make", "jsonnet")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range targets {
		f1 := filepath.Join(dir, f.Name())
		f2 := filepath.Join("testdata", f.Name())
		assertFileEqual(t, f1, f2)
	}
}
