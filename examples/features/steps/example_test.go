
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
func init() {
    Step(`a new user visits the gorkin site`, func() {
        fmt.Println("visit the gorkin site here...")
    })

    Step(`they should see an example`, func(t *T) {
        t.Log("test that they saw the example.")
    })
}

func Test(t *T) {
    // RunFeatureTests registers Go's testing runner and your
    // isolation type with Gorkin and then runs the defined steps
    // against any feature files found.
    RunFeatureTests(t, &I{})
}
