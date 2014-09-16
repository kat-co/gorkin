package main

import (
	"bufio"
	"fmt"
	. "github.com/katco-/vala"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"text/tabwriter"
)

var debug *log.Logger

func init() {
	var debugBuff io.Writer
	if true {
		debugBuff = os.Stderr
	} else {
		debugBuff = ioutil.Discard
	}
	debug = log.New(debugBuff, "DEBUG: ", log.Llongfile)
}

func main() {

}

func RunFeatureTests(t *testing.T, stepIsolater interface{}) {
	stepType := reflect.PtrTo(reflect.TypeOf(stepIsolater)).Elem()
	if files, err := ioutil.ReadDir("features"); err != nil {
		log.Fatalf("could not read features directory: %v", err)
	} else {

		for _, f := range files {

			fmt.Printf("Processing: \"%s\"\n\n", f.Name())
			if filepath.Ext(f.Name()) != ".feature" {
				continue
			} else if feat, err := ioutil.ReadFile(filepath.Join("features", f.Name())); err != nil {
				log.Fatalf("could not read feature file: %v", err)
			} else {
				if r, err := handleFeature(bufio.NewReader(strings.NewReader(string(feat)))); err != nil {
					log.Fatalf("%v", err)
				} else if err := run(r, t, stepType); err != nil {
					log.Fatalf("%v", err)
				}
			}
		}
	}
}

func handleFeature(fReader *bufio.Reader) (runners []*runnerAndArgs, err error) {
	BeginValidation().Validate(IsNotNil(fReader, "fReader")).CheckAndPanic()

	// TODO(kate): Stupid dumb parsing just to get things moving.

	type mode string

	const (
		GivenMode       mode = "Given"
		WhenMode        mode = "When"
		ThenMode        mode = "Then"
		DeclarationMode mode = "(declaration)"
	)

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
	lastMode := DeclarationMode
	atLeastOneMissingRunner := false

	for {
		var line string
		if line, err = fReader.ReadString('\n'); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		} else {
			line = strings.TrimSpace(line)
		}

		andSpecified := strings.HasPrefix(line, "And")
		andModeFor := func(mode mode) bool {
			//debug.Printf("andMode=%v, lastMode=%v, mode=%v", andSpecified, lastMode, mode)
			return andSpecified && lastMode == mode
		}

		switch {
		default:
			return nil, fmt.Errorf("Unknown line type: %s", strings.Split(line, " ")[0])
		case line == "":
			break
		case strings.HasPrefix(line, "Feature:"):
			lastMode = DeclarationMode
			_, err = ParseFeature(line, fReader)
		case strings.HasPrefix(line, "Background:"):
			lastMode = DeclarationMode
		case strings.HasPrefix(line, "Scenario"):
			lastMode = DeclarationMode
			//handleScenarioBlock(fReader)
			err = fmt.Errorf("Not implemented.")
		case strings.HasPrefix(line, "Given") || andModeFor(GivenMode):
			lastMode = GivenMode
			if runner, err := ParseGiven(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, "When") || andModeFor(WhenMode):
			lastMode = WhenMode
			if runner, err := ParseWhen(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, "Then") || andModeFor(ThenMode):
			lastMode = ThenMode
			if runner, err := ParseThen(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				runners = append(runners, runner)
			}
		case andSpecified:
			return nil, fmt.Errorf("and clauses may only follow a Given, When, or Then clause.")
		}

		// Let user know status of line
		fmt.Fprintf(w, line)
		if err != nil {
			fmt.Fprintf(w, "\t// %s", err)
		}
		fmt.Fprintln(w)
	}

	w.Flush()

	if atLeastOneMissingRunner {
		return nil, fmt.Errorf("Please implement the missing runners.")
	}

	return runners, nil
}

type runnerAndArgs struct {
	Runner runner
	Args   []string
	Step   string
}

func run(runners []*runnerAndArgs, t *testing.T, stepType reflect.Type) error {

	tType := reflect.TypeOf(t) // Did I stutter?
	stepVal := reflect.New(stepType.Elem())

	for _, r := range runners {
		rt := reflect.TypeOf(r.Runner)
		if rt.Kind() != reflect.Func {
			return fmt.Errorf("Steps must be functions, not %v", rt)
		}

		numGorkinGeneratableTypes := 0
		if numOff := int(math.Abs(float64(rt.NumIn() - len(r.Args)))); numOff != 0 {

			// Loop through the arguments to see if we can't explain
			// the difference by types we know how to create.
			for pn := 0; pn < rt.NumIn(); pn++ {
				switch rt.In(pn) {
				case tType, stepType:
					numGorkinGeneratableTypes += 1
					// default:
					// 	t.Logf("Failed type: %v", rt.In(pn))
				}
			}

			if numGorkinGeneratableTypes < numOff {
				t.Fatalf(
					"Regex did not provide enough groups (%d) for given step (%d):\n\t%s",
					rt.NumIn(),
					len(r.Args),
					r.Step,
				)
				return nil
			}
		}

		// Build up the arguments
		var args []reflect.Value
		for pn, an := 0, 0-numGorkinGeneratableTypes; pn < rt.NumIn(); pn, an = pn+1, an+1 {

			// if an >= len(r.Args) {
			// 	t.Fatalf(
			// 		"Regex did not provide enough groups (%d) for given step (%d):\n\t%s",
			// 		rt.NumIn(),
			// 		len(r.Args),
			// 		r.Step,
			// 	)
			// } else {
			// 	t.Logf("an: %d", an)
			// }

			switch rt.In(pn) {
			default:
				return fmt.Errorf(`Cannot handle steps which accept arguments of type "%v" at this time.`, rt.In(pn))
			case stepType:
				args = append(args, stepVal)
			case tType:
				args = append(args, reflect.ValueOf(t))
			case reflect.TypeOf(true):
				b := false
				if r.Args[an] != "" {
					b = true
				}
				args = append(args, reflect.ValueOf(b))
			case reflect.TypeOf(""):
				args = append(args, reflect.ValueOf(r.Args[an]))
			}
		}

		reflect.ValueOf(r.Runner).Call(args)
	}
	return nil
}

func readFeatureFile(f os.FileInfo) (string, error) {
	BeginValidation().Validate(IsNotNil(f, "f")).CheckAndPanic()

	if c, err := ioutil.ReadFile(f.Name()); err != nil {
		return "", fmt.Errorf("could not read feature file: %v", err)
	} else {
		return string(c), nil
	}
}
