package steps

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"os/exec"

	. "github.com/kat-co/gorkin/gorkin"
)

func TestFeatures(t *testing.T) {

	// Isolation layer.
	type I struct {
		dir         string
		featureFile string
		gorkResult  string
	}

	Step(`the path \"([^"]+)\"( doesn't)? exists?`, func(f *I, dirName string, deleteIfExists bool) {

		// Create a temporary directory to operate within.
		if f.dir == "" {
			isolationDir, err := ioutil.TempDir("", "gorkin-test")
			if err != nil {
				t.Fatalf("could not create tmp host directory: %s", err)
			}
			f.dir = isolationDir
		}

		if newDir := filepath.Join(f.dir, dirName); deleteIfExists == false {
			if err := os.Mkdir(newDir, 0777); err != nil {
				t.Fatalf("could not create directory: %s", err)
			}
		} else if err := os.RemoveAll(newDir); err != nil {
			t.Fatalf("could not remove directory: %v", err)
		}
	})

	Step(`the file \"([^"]+)\" exists(?: with content)?`, func(f *I, filePath, content string) {

		file := filepath.Join(f.dir, filePath)
		if err := ioutil.WriteFile(file, []byte(content), 0666); err != nil {
			t.Fatalf("could not create feature file: %v", err)
		}
	})

	Step(`there is a steps directory under features`, func(f *I) {
		if err := os.Mkdir(filepath.Join(f.dir, "features", "steps"), 0777); err != nil {
			t.Fatalf("could not create feature file: %v", err)
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

	Step(`the output should be`, func(f *I, output string) {
		if f.gorkResult != output {
			t.Errorf(`Expected output: "%s"`, output)
			t.Fatalf(`unexpected result from gorkin: "%v"`, f.gorkResult)
		}
	})

	Step(`the output should contain`, func(f *I, output string) {
		if strings.Contains(output, f.gorkResult) {
			t.Logf(`Expected output: "%s"`, output)
			t.Fatalf(`unexpected result from gorkin: "%v"`, f.gorkResult)
		}
	})

	RunFeatureTests(t, &I{})
}
