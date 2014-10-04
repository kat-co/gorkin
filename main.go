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
	"runtime"
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
	files, err := ioutil.ReadDir("features")
	if err != nil {
		log.Fatalf("could not read features directory: %v", err)
	}

	for _, f := range files {

		fmt.Printf("Processing: \"%s\"\n\n", f.Name())
		if filepath.Ext(f.Name()) != ".feature" {
			continue
		}

		feat, err := ioutil.ReadFile(filepath.Join("features", f.Name()))
		if err != nil {
			log.Fatalf("could not read feature file: %v", err)
		}

		if r, err := handleFeature(bufio.NewReader(strings.NewReader(string(feat)))); err != nil {
			t.Errorf("\n\n%v", err)
			return
		} else if err := run(r, t, stepType); err != nil {
			log.Fatalf("%v", err)
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
		PythonStrSent   mode = "PythonSent"
		PythonString    mode = "Python"
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

		var runner *runnerAndArgs
		var lineComment string

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
			if runner, err = ParseGiven(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, "When") || andModeFor(WhenMode):
			lastMode = WhenMode
			if runner, err = ParseWhen(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, "Then") || andModeFor(ThenMode):
			lastMode = ThenMode
			if runner, err = ParseThen(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, `"""`) || lastMode == PythonString:
			lastRunner := &runners[len(runners)-1]
			args := (*lastRunner).Args
			if strings.HasPrefix(line, `"""`) {
				if lastMode != PythonString {
					args = append(args, "")
				} else {
					lastArg := &args[len(args)-1]
					*lastArg = (*lastArg)[:len((*lastArg))-1]
				}
				(*lastRunner).Args = args
				lastMode = PythonString
				break
			}

			lastArg := &args[len(args)-1]
			*lastArg += line + "\n"
			(*lastRunner).Args = args
		case andSpecified:
			return nil, fmt.Errorf("and clauses may only follow a Given, When, or Then clause.")
		}

		// Let user know status of line
		fmt.Fprintf(w, line+"\t")
		if err != nil {
			fmt.Fprintf(w, "✗ %s", err)
		} else if lineComment != "" {
			fmt.Fprintf(w, "✓ %s", lineComment)
		}

		fmt.Fprintf(w, "\n")
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

func (r *runnerAndArgs) StepInfo() string {
	fp := reflect.ValueOf(r.Runner).Pointer()
	f, l := runtime.FuncForPC(fp).FileLine(fp)

	if wd, err := os.Getwd(); err != nil {
		panic(err)
	} else if relPath, err := filepath.Rel(wd, f); err != nil {
		panic(err)
	} else {
		return fmt.Sprintf("%s:%d", relPath, l)
	}
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
					numGorkinGeneratableTypes++
					// default:
					// 	t.Logf("Failed type: %v", rt.In(pn))
				}
			}

			if numGorkinGeneratableTypes < numOff {
				t.Fatalf(
					"Regex provided %d groups, but the step requires %d arguments:\n\t%s",
					len(r.Args),
					rt.NumIn(),
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
