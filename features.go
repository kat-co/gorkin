package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Steps are global for maximum reusability.
var (
	givenRunners runnerMap = make(map[*regexp.Regexp]runner)
	whenRunners            = make(map[*regexp.Regexp]runner)
	thenRunners            = make(map[*regexp.Regexp]runner)
)

func ParseFeature(featureLine string, reader *bufio.Reader) (*Feature, error) {

	description := strings.TrimSpace(strings.TrimLeft(featureLine, "Feature:"))
	if description == "" {
		return nil, fmt.Errorf("Please provide a description for this feature.")
	}

	caseStatement := new(bytes.Buffer)
	for {
		if line, err := reader.ReadString('\n'); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		} else {
			if line = strings.TrimSpace(line); line == "" {
				break
			}
			fmt.Fprintf(caseStatement, "\n%s", line)
		}
	}

	return &Feature{
		description:   description,
		caseStatement: caseStatement.String(),
	}, nil
}

type Feature struct {
	description   string
	caseStatement string
	background    Background
	scenarios     []*Scenario
}

func ParseBackground(backgroundLine string, reader *bufio.Reader) (Background, error) {
	return Background{}, fmt.Errorf("Not implemented.")
}

type runner interface{}
type runnerMap map[*regexp.Regexp]runner

type Background struct {
	givenRunners runnerMap
}

func ParseGiven(givenLine string, reader *bufio.Reader) (*runnerAndArgs, error) {
	return findRunner(strings.TrimLeft(givenLine, "Given "), givenRunners, reader)
}

func Given(regex string, f runner) {
	must(runnerExists("Given", regex, givenRunners))
	givenRunners[regexp.MustCompile(regex)] = f
}

func UsingScenario(description string) *Scenario {
	return &Scenario{
		description: description,
		// Background:  Background{make(map[string]runner)},
		// whenRunners: make(map[string]runner),
		// thenRunners: make(map[string]runner),
	}
}

type Scenario struct {
	Background
	description string
	whenRunners runnerMap
	thenRunners runnerMap
}

func ParseWhen(whenLine string, reader *bufio.Reader) (*runnerAndArgs, error) {
	return findRunner(strings.TrimLeft(whenLine, "When "), whenRunners, reader)
}

func When(regex string, f runner) {
	must(runnerExists("When", regex, whenRunners))
	whenRunners[regexp.MustCompile(regex)] = f
}

func ParseThen(thenLine string, reader *bufio.Reader) (*runnerAndArgs, error) {
	return findRunner(strings.TrimLeft(thenLine, "Then "), thenRunners, reader)
}

func Then(regex string, f runner) {
	must(runnerExists("Then", regex, thenRunners))
	thenRunners[regexp.MustCompile(regex)] = f
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func runnerExists(rType string, regex string, runners runnerMap) error {

	for cRegex, _ := range runners {
		if cRegex.String() == regex {
			return fmt.Errorf(`A "%s" runner for "%s" already exists.`, rType, regex)
		}
	}
	return nil
}

func findRunner(line string, runners runnerMap, reader *bufio.Reader) (found *runnerAndArgs, err error) {

	for cRegex, runner := range runners {

		if matches := cRegex.FindStringSubmatch(line); len(matches) != 0 {
			if found != nil {
				// TODO(kate): Give more information.
				// TODO(kate): Could we detect this during definition?
				// e.g. If one regex is a subset of another?
				return nil, fmt.Errorf("Conflicting runners!")
			}

			// Elide the master match at index 0.
			found = &runnerAndArgs{runner, matches[1:], cRegex.String()}
		}
	}

	if found == nil {
		err = fmt.Errorf("No matching runner.")
	}

	return found, err
}
