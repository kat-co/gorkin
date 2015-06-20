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

		if f, err := handleFeature(bufio.NewReader(strings.NewReader(string(feat)))); err != nil {
			t.Errorf("\n\n%v", err)
			return
		} else if _, err := run(f.Runners, t, stepType, f.Background...); err != nil {
			log.Fatalf("%v", err)
		}
	}

	if numFeaturesProcessed <= 0 {
		log.Fatal("No feature files found.")
	}
}

func handleFeature(fReader *bufio.Reader) (ftr *feature, err error) {
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
		BackgroundMode  mode = "Background"
	)

	ftr = &feature{}
	w := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
	modeStack := []mode{DeclarationMode}
	atLeastOneMissingRunner := false
	scenarioIndentation := 0
	pythonString := ""

	endOfBackgroundBlock := func(stateStack []mode) ([]*runnerAndArgs, bool) {
		backgroundRunners := make([]*runnerAndArgs, 0)
		for posFromHead, state := range stateStack {
			//debug.Printf("Adding: %v", (*ftr.Runners[len(ftr.Runners)-posFromHead-1]))
			switch state {
			case BackgroundMode:

				debug.Println("== All steps: ==")
				debugSteps(ftr.Runners...)
				debug.Println("== / ==")

				// Remove all the background runners from the normal
				// stack.
				pivotPoint := len(ftr.Runners) - posFromHead - 1
				// TODO(katco): For some reason, copy is not working. Probably because the array contains pointers to structs.
				for _, runner := range ftr.Runners[pivotPoint:] {
					backgroundRunners = append(backgroundRunners, runner)
				}
				//backgroundRunners = ftr.Runners[pivotPoint:]
				debug.Println("== backgroundRunners: ==")
				debugSteps(backgroundRunners...)
				debug.Println("== / ==")
				ftr.Runners = ftr.Runners[:pivotPoint]
				debug.Println("== All steps: ==")
				debugSteps(ftr.Runners...)
				debug.Println("== / ==")
				return backgroundRunners, true
			case DeclarationMode:
				return nil, false
			}
		}
		return nil, false
	}

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
			return andSpecified && modeStack[0] == mode
		}

		// debug.Printf(`Processing line (mode:%s): "%s"\\n`, modeStack[0], line)
		// debug.Printf(`Processing rawLine: "%s"\\n`, rawLine)

		var runner *runnerAndArgs
		var lineComment string

		switch {
		default:
			return nil, fmt.Errorf("Unknown line type: %s", strings.Split(line, " ")[0])
		case strings.HasPrefix(line, `"""`):

			if modeStack[0] == PythonString {
				// Pop PythonString if it has ended
				modeStack = modeStack[1:]

				if len(ftr.Runners) > 0 {
					args := ftr.Runners[len(ftr.Runners)-1].Args
					if len(pythonString) != 0 {
						pythonString = pythonString[1:]
					}
					args = append(args, fmt.Sprintf(pythonString))
					ftr.Runners[len(ftr.Runners)-1].Args = args
				}

				pythonString = ""
			} else {
				// Push PythonString
				scenarioIndentation = indentCount
				modeStack = append([]mode{PythonString}, modeStack...)
			}
		case modeStack[0] == PythonString && len(ftr.Runners) > 0:
			// The python string we're building up is meant to be an
			// argument in the runner which proceded it. If there
			// aren't any runners, it's just some helpful python
			// string, but not an argument.

			pythonString += "\n"
			if len(rawLine) > 1 {
				pythonString += rawLine[scenarioIndentation-1 : len(rawLine)-1]
			}
		case line == "":
			if len(ftr.Background) <= 0 {
				if backgroundRunners, ok := endOfBackgroundBlock(modeStack); ok {
					ftr.Background = backgroundRunners
				}
			}
			break
		case strings.HasPrefix(line, "Feature:"):
			modeStack = append([]mode{DeclarationMode}, modeStack...)
			_, err = ParseFeature(line, fReader)
		case strings.HasPrefix(line, "Background:"):

			if ftr.Background != nil {
				err = fmt.Errorf("multiple backgrounds defined")
			}

			modeStack = append([]mode{BackgroundMode}, modeStack...)
			ftr.Runners = append(ftr.Runners, &runnerAndArgs{Step: line})
		case strings.HasPrefix(line, "Scenario"):

			if len(ftr.Background) <= 0 {
				if backgroundRunners, ok := endOfBackgroundBlock(modeStack); ok {
					ftr.Background = backgroundRunners
				}
			}

			modeStack = append([]mode{DeclarationMode}, modeStack...)
			ftr.Runners = append(ftr.Runners, &runnerAndArgs{Step: line})
		case strings.HasPrefix(line, "Given") || andModeFor(GivenMode):
			modeStack = append([]mode{GivenMode}, modeStack...)
			if runner, err = ParseGiven(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				ftr.Runners = append(ftr.Runners, runner)
			}
		case strings.HasPrefix(line, "When") || andModeFor(WhenMode):
			modeStack = append([]mode{WhenMode}, modeStack...)
			if runner, err = ParseWhen(line, fReader); err != nil {
				atLeastOneMissingRunner = true
			} else {
				lineComment = runner.StepInfo()
				ftr.Runners = append(ftr.Runners, runner)
			}
		case strings.HasPrefix(line, "Then") || andModeFor(ThenMode):
			modeStack = append([]mode{ThenMode}, modeStack...)
			runner, err = ParseThen(line, fReader)
			if err != nil {
				atLeastOneMissingRunner = true
				break
			}
			lineComment = runner.StepInfo()
			ftr.Runners = append(ftr.Runners, runner)
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

	debug.Printf("Feature has %d background steps, and %d other steps",
		len(ftr.Background),
		len(ftr.Runners),
	)
	debug.Println("== Background steps: ==")
	debugSteps(ftr.Background...)
	debug.Println("== / ===")

	return ftr, nil
}

func debugSteps(steps ...*runnerAndArgs) {
	for _, step := range steps {
		debug.Println(step.Step)
	}
}

// feature contains everything needed to execute tests against a
// feature file.
type feature struct {
	// Background represents a defined background clause for the
	// feature. This will be executed before each Scenario.
	Background []*runnerAndArgs

	// Runners represent all the lines of the feature and the
	// corresponding tests.
	Runners []*runnerAndArgs
}

type runnerAndArgs struct {
	// Runner is the function to run.
	Runner runner
	// Args contains the string representations of the arguments to
	// the runner.
	Args []string
	// Step is the line in the feature file which was matched.
	Step string
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
) func(string, int, int, func(int) reflect.Type) error {
	return func(step string, numStepParams, numRegexGroups int, paramTypeFn func(int) reflect.Type) error {
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
				numStepParams-numGorkinGeneratableTypes,
				step,
			)
		}

		return nil
	}
}

func run(runners []*runnerAndArgs, t *testing.T, stepType reflect.Type, backgroundSteps ...*runnerAndArgs) (reflect.Value, error) {

	debug.Printf("Running %d steps with %d background steps.",
		len(runners),
		len(backgroundSteps),
	)

	tType := reflect.TypeOf(t)
	stepVal := reflect.New(stepType.Elem())
	accountForParamAndArgDiff := accountForParamAndArgDiffFn(t, stepType)

	for _, r := range runners {

		debug.Printf(`Processing step: "%v"`, r.Step)

		if strings.HasPrefix(r.Step, "Scenario") {
			fmt.Println(r.Step)
			// For each scenario, re-run the background clause.
			debug.Println("== BACKGROUND ==")
			if newContext, err := run(backgroundSteps, t, stepType); err != nil {
				return stepVal, err
			} else {
				debug.Println("== END BACKGROUND ==")
				stepVal = newContext
			}
			continue
		} else if strings.HasPrefix(r.Step, "Background") {
			continue
		}

		rt := reflect.TypeOf(r.Runner)
		if rt.Kind() != reflect.Func {
			return stepVal, fmt.Errorf("Steps must be functions, not %v", rt)
		}

		numRegexArgs := len(r.Args)
		numStepArgs := rt.NumIn()
		if numStepArgs > numRegexArgs {
			err := accountForParamAndArgDiff(r.Step, numStepArgs, numRegexArgs, rt.In)
			if err != nil {
				fmt.Printf("WARNING: %v\n", err)
				//return stepVal, err
			}
		}

		// Build up the arguments
		var args []reflect.Value
		for stepArgIdx, regexGroupIdx := 0, 0; stepArgIdx < numStepArgs; stepArgIdx++ {

			switch rt.In(stepArgIdx) {
			default:
				return stepVal, fmt.Errorf(
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
				if len(r.Args) > regexGroupIdx {
					args = append(args, reflect.ValueOf(r.Args[regexGroupIdx]))
				} else {
					fmt.Println("WARNING: Assuming a doc string will be passed in.")
					args = append(args, reflect.ValueOf(""))
				}
				regexGroupIdx++
			case reflect.TypeOf(0):
				i, err := strconv.Atoi(r.Args[regexGroupIdx])
				if err != nil {
					return stepVal, err
				}
				args = append(args, reflect.ValueOf(i))
				regexGroupIdx++
			}
		}

		reflect.ValueOf(r.Runner).Call(args)
	}
	return stepVal, nil
}

func readFeatureFile(f os.FileInfo) (string, error) {
	BeginValidation().Validate(IsNotNil(f, "f")).CheckAndPanic()

	if c, err := ioutil.ReadFile(f.Name()); err != nil {
		return "", fmt.Errorf("could not read feature file: %v", err)
	} else {
		return string(c), nil
	}
}
