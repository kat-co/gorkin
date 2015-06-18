package steps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/kat-co/gorkin/gorkin"
	"os/exec"
)

func TestFeatures(t *testing.T) {

	// Isolation layer.
	type I struct {
		dir         string
		featureFile string
		gorkResult  string
	}

	Step(`there is( not)?? a directory named \"(\w+)\"`, func(f *I, noDir bool, dirName string) {

		// Create a temporary directory to operate within.
		dirName, err := ioutil.TempDir("", "gorkin-test")
		if err != nil {
			t.Fatalf("could not create tmp host directory: %s", err)
		}
		f.dir = dirName

		if !noDir {
			if err := os.Mkdir(filepath.Join(dirName, "features"), 0777); err != nil {
				t.Fatalf("could not create directory: %s", err)
			}
		}
	})

	Step(`there is at least 1 feature file`, func(f *I) {

		f.featureFile = filepath.Join(f.dir, "features", "foo.feature")
		if _, err := os.Create(f.featureFile); err != nil {
			t.Fatalf("could not create feature file: %v", err)
		}
	})

	Step(`there is a steps directory under features`, func(f *I) {
		if err := os.Mkdir(filepath.Join(f.dir, "features", "steps"), 0777); err != nil {
			t.Fatalf("could not create feature file: %v", err)
		}	
	})

	Step(`there is at least 1 gorkin test file`, func(f *I) {
		testFile := filepath.Join(f.dir, "features", "steps", "foo_test.go")
		err := ioutil.WriteFile(testFile, []byte(
			`package steps

import (
	"testing"

	. "github.com/kat-co/gorkin/gorkin"
)

func Test(t *testing.T) {

	type I struct {}

	RunFeatureTests(t, &I{})	
}`), 0666)
		if err != nil {
			t.Fatalf("could not write test file: %v", err)
		}
	})

	Step(`a user runs gorkin`, func(f *I) {

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("could not get cwd: %v", err)
		}
		defer func() {
			os.Chdir(cwd)
		}()
		if err := os.Chdir(f.dir); err != nil {
			t.Fatalf("could not change into temp directory: %v\n", err)
		}

		cmd := exec.Command("gorkin")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("could not run gorkin: %v", err)
		}

		f.gorkResult = string(output)
	})

	Step(`gorkin should find the features directory`, func(f *I) {
		if strings.Contains(f.gorkResult, "Processing:") == false {
			t.Fatalf("gorkin did not find the features directory: %s", f.gorkResult)
		}
	})

	Step(`it should find the .feature files within that directory`, func(f *I) {
		fmt.Println("checking to see if gorkin found the .feature file in that directory.")
		if f.featureFile == "" {
			t.Fatal("Did not find a .feature file.")
		}
	})

	Step(`they should receive this error`, func(f *I, errMsg string) {
		if f.gorkResult != errMsg {
			t.Fatalf(`unexpected result from gorkin: "%v"`, f.gorkResult)
		}
	})

	Step("a gorkin feature with 2 scenarios", func() {
		t.Fatal("Not implemented.")
	})

	Step("a step which conflicts with another", func() {
		t.Fatal("Not implemented")
	})

	Step("the steps should not conflict with one another", func() {
		t.Fatal("Not implemented.")
	})

	RunFeatureTests(t, &I{})
}
