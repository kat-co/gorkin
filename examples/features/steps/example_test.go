
package example

import (
    . "testing"
    "fmt"

    . "github.com/kat-co/gorkin/gorkin"
)

// I is a structure designed to isolate testing state. For each
// scenario, a new one of these will be instantiated and passed to
// test steps when requested.
type I struct {}

// Steps can be registered anywhere. The only stipulation is that they
// all be registered before the call to RunFeatureTests is made. Since
// step registration may need to be spread across several files, the
// init function is a good choice.
//
// The step's function signature can also be whatever you'd like it to
// be. In most cases, gorkin will do the right thing.
func init() {
    Step(`a new user visits the gorkin site`, func() {
        fmt.Println("visit the gorkin site here...")
    })

    // Here we see an example of how to match an element of a
    // scenario's step so that we can utilize it in the step's
    // function definition. gorkin will automatically perform the type
    // conversion.
    Step(`they should see ([0-9]+) example(?:s)?`, func(t *T, numExamples int) {
        t.Logf("user should see %d", numExamples)
    })
}

func Test(t *T) {
    // RunFeatureTests registers Go's testing runner and your
    // isolation type with Gorkin and then runs the defined steps
    // against any feature files found.
    RunFeatureTests(t, &I{})
}
