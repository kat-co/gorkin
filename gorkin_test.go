package main

import (
	"fmt"
	"testing"
)

func TestFeatures(t *testing.T) {

	// Isolation layer.
	type I struct {
		dir         string
		featureFile string
		gorkResult  string
	}

	Given(`there is( not)?? a directory named \"(\w+)\"`, func(f *I, noDir bool, dirName string) {

		if !noDir {
			f.dir = dirName
			fmt.Printf("created directory %s\n", f.dir)
		} else {
			f.dir = ""
			fmt.Printf(`This should be empty: "%s"`+"\n", f.dir)
		}
	})

	Given(`there is at least 1 feature file`, func(f *I) {
		f.featureFile = "foo.feature"
		fmt.Println("created a feature file")
	})

	When(`the user runs gorkin`, func(f *I) {
		if f.dir != "features" {
			f.gorkResult = "error"
		} else {
			f.gorkResult = "ran"
		}

		fmt.Printf("ran gorkin: %v\n", f.gorkResult)
	})

	Then(`gorkin should find the features directory`, func(f *I) {
		fmt.Println("checking to see if gorkin found the features directory")
		if f.dir != "features" {
			t.Fatalf("Did not find the features directory.")
		}
	})

	Then(`it should find the .feature files within that directory`, func(f *I) {
		fmt.Println("checking to see if gorkin found the .feature file in that directory.")
		if f.featureFile == "" {
			t.Fatal("Did not find a .feature file.")
		}
	})

	Then(`they should receive an error`, func(f *I) {
		fmt.Println("checking to make sure there is an error.")
		if f.gorkResult != "error" {
			t.Fatalf("Expected to receive an error: %v", f.gorkResult)
		}
	})

	Given("a gorkin feature with 2 scenarios", func() {
	})

	Given("a step which conflicts with another", func() {
	})

	Then("the steps should not conflict with one another", func() {
		//t.Error("Not implemented.")
	})

	RunFeatureTests(t, &I{})
}
