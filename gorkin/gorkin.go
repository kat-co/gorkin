package gorkin

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"text/tabwriter"

	. "github.com/kat-co/vala"
)

func init() {
	var debugBuff io.Writer
	if false {
		debugBuff = os.Stderr
	} else {
		debugBuff = ioutil.Discard
	}
	debug = log.New(debugBuff, "DEBUG: ", log.Llongfile)
}

var debug *log.Logger

func RunFeatureTests(t *testing.T, stepIsolater interface{}) {
	stepType := reflect.PtrTo(reflect.TypeOf(stepIsolater)).Elem()
	// Steps should be in the steps folder under features
	files, err := ioutil.ReadDir("../")
	if err != nil {
		log.Fatalf("could not read features directory: %v", err)
	}

	numFeaturesProcessed := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) != ".feature" {
			continue
		}
		numFeaturesProcessed++

		fmt.Println(os.Getwd())
		fmt.Printf("Processing: \"%s\".\n", f.Name())
		feat, err := ioutil.ReadFile(filepath.Join("..", f.Name()))
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

	if numFeaturesProcessed <= 0 {
		log.Fatal("No feature files found.")
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
		PythonString    mode = "Python"
		Example         mode = "Example"
		DeclarationMode mode = "(declaration)"
	)

	w := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
	modeStack := []mode{DeclarationMode}
	atLeastOneMissingRunner := false
	scenarioIndentation := 0

	for {
		var rawLine string
		if rawLine, err = fReader.ReadString('\n'); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		line := strings.TrimSpace(rawLine)
		indentCount := len(rawLine) - len(line)
		andSpecified := strings.HasPrefix(line, "And")
		andModeFor := func(mode mode) bool {
			//debug.Printf("andMode=%v, lastMode=%v, mode=%v", andSpecified, lastMode, mode)
			return andSpecified && modeStack[0] == mode
		}

		//debug.Printf("Processing line (mode:%s): %s\n", modeStack[0], line)

		var runner *runnerAndArgs
		var lineComment string

		switch {
		default:
			return nil, fmt.Errorf("Unknown line type: %s", strings.Split(line, " ")[0])
		case line == "":
			break
		case strings.HasPrefix(line, "Feature:"):
			modeStack = append([]mode{DeclarationMode}, modeStack...)
			_, err = ParseFeature(line, fReader)
		case strings.HasPrefix(line, "Background:"), strings.HasPrefix(line, "Scenario"):
			modeStack = append([]mode{DeclarationMode}, modeStack...)
			runners = append(runners, &runnerAndArgs{Step: line})
		case strings.HasPrefix(line, "Given") || andModeFor(GivenMode):
			modeStack = append([]mode{GivenMode}, modeStack...)
			if runner, err = ParseGiven(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, "When") || andModeFor(WhenMode):
			modeStack = append([]mode{WhenMode}, modeStack...)
			if runner, err = ParseWhen(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				runners = append(runners, runner)
			}
		case strings.HasPrefix(line, "Then") || andModeFor(ThenMode):
			modeStack = append([]mode{ThenMode}, modeStack...)
			runner, err = ParseThen(line, fReader)
			if err != nil {
				atLeastOneMissingRunner = true
				break
			}
			lineComment = runner.StepInfo()
			runners = append(runners, runner)
		case strings.HasPrefix(line, `"""`) || modeStack[0] == PythonString:
			if len(runners) <= 0 {
				if strings.HasPrefix(line, `"""`) && modeStack[0] == PythonString {
					modeStack = modeStack[1:]
					break
				}
				scenarioIndentation = indentCount
				modeStack = append([]mode{PythonString}, modeStack...)
				break
			}
			lastRunner := func() *runnerAndArgs { return runners[len(runners)-1] }
			args := lastRunner().Args
			if strings.HasPrefix(line, `"""`) {
				if modeStack[0] != PythonString {
					args = append(args, "")
					scenarioIndentation = indentCount
					modeStack = append([]mode{PythonString}, modeStack...)
				} else {
					lastArg := &args[len(args)-1]
					*lastArg = (*lastArg)[:len((*lastArg))-1]
					modeStack = modeStack[1:]
				}
				lastRunner().Args = args
				break
			}

			lastArg := &args[len(args)-1]
			*lastArg += rawLine[scenarioIndentation-1:]
			lastRunner().Args = args
		case strings.HasPrefix(line, "Examples:") || modeStack[0] == Example:
			modeStack = append([]mode{Example}, modeStack...)
			err = fmt.Errorf("Not yet supported")
		case andSpecified:
			fmt.Errorf("for line: %s", line)
			return nil, fmt.Errorf("and clauses may only follow a Given, When, or Then clause.")
		}

		// Let user know status of line
		fmt.Fprintf(w, strings.Repeat("  ", indentCount)+line+"\t")
		if err != nil {
			fmt.Fprintf(w, "✗ %s", err)
		} else if lineComment != "" {
			fmt.Fprintf(w, "✓ %s", lineComment)
		}

		fmt.Fprintf(w, "\n")
	}

	if atLeastOneMissingRunner {
		w.Flush()
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

func accountForParamAndArgDiffFn(
	forTest *testing.T,
	forStep reflect.Type,
) func(int, int, func(int) reflect.Type) error {
	return func(numStepParams, numRegexGroups int, paramTypeFn func(int) reflect.Type) error {
		numOff := numStepParams - numRegexGroups

		// Loop through the arguments to see if we can't explain
		// the difference by types we know how to create.
		numGorkinGeneratableTypes := 0
		for pn := 0; pn < numStepParams; pn++ {
			switch paramTypeFn(pn) {
			case reflect.TypeOf(forTest), forStep:
				numGorkinGeneratableTypes++
				// default:
				// 	t.Logf("Failed type: %v", rt.In(pn))
			}
		}

		if numGorkinGeneratableTypes < numOff {
			return fmt.Errorf(
				"Regex provided %d groups, but the step requires %d arguments:\n\t%s",
				numRegexGroups,
				numStepParams,
				"Foo",
				//r.Step,
			)
		}

		return nil
	}
}

func run(runners []*runnerAndArgs, t *testing.T, stepType reflect.Type) error {

	tType := reflect.TypeOf(t) // Did I stutter?
	stepVal := reflect.New(stepType.Elem())
	accountForParamAndArgDiff := accountForParamAndArgDiffFn(t, stepType)

	for _, r := range runners {

		debug.Printf("Runner: %v", r)

		if strings.HasPrefix(r.Step, "Scenario") || strings.HasPrefix(r.Step, "Background") {
			//debug.Printf("IN HERE")
			stepVal = reflect.New(stepType.Elem())
			continue
		}

		rt := reflect.TypeOf(r.Runner)
		if rt.Kind() != reflect.Func {
			return fmt.Errorf("Steps must be functions, not %v", rt)
		}

		numRegexArgs := len(r.Args)
		numStepArgs := rt.NumIn()
		if numStepArgs > numRegexArgs {
			err := accountForParamAndArgDiff(numStepArgs, numRegexArgs, rt.In)
			if err != nil {
				return err
			}
		}

		// Build up the arguments
		var args []reflect.Value
		for stepArgIdx, regexGroupIdx := 0, 0; stepArgIdx < numStepArgs; stepArgIdx++ {

			switch rt.In(stepArgIdx) {
			default:
				return fmt.Errorf(
					`Cannot handle steps which accept arguments of type "%v" at this time.`,
					rt.In(stepArgIdx),
				)
			case tType:
				args = append(args, reflect.ValueOf(t))
			case stepType:
				args = append(args, stepVal)
			case reflect.TypeOf(true):
				b := false
				if r.Args[regexGroupIdx] != "" {
					b = true
				}
				args = append(args, reflect.ValueOf(b))
				regexGroupIdx++
			case reflect.TypeOf(""):
				args = append(args, reflect.ValueOf(r.Args[regexGroupIdx]))
				regexGroupIdx++
			case reflect.TypeOf(0):
				i, err := strconv.Atoi(r.Args[regexGroupIdx])
				if err != nil {
					return err
				}
				args = append(args, reflect.ValueOf(i))
				regexGroupIdx++
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
