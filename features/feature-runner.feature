Feature: Running Over Feature Files
  As a gorkin user
  I would like gorkin to run tests against feature files
  So that I can couple my requirements with my tests.

  Background:
    Given the path "./features" exists
    And the path "./features/steps" exists

  Scenario: A user runs gorkin without a features directory.
    Given the path "./features" doesn't exist
    When a user runs "gorkin"
    Then the output should be
    """
    could not find a features directory.
    """

  Scenario: A user runs gorkin with feature files and steps.
    Given the file "./features/foo.feature" exists
    And the file "./features/steps/foo_test.go" exists with content
    """
    package steps

    import (
        "testing"

        . "github.com/kat-co/gorkin/gorkin"
    )

    func Test(t *testing.T) {
        type I struct {}
        RunFeatureTests(t, &I{})
    }
    """
    When a user runs "gorkin"
    Then the output should contain
    """
    ok
    """


  Scenario: A user runs multiple scenarios with reusable steps.
    Given the file "./features/reusable-steps.feature" exists with content
    """
    Feature: Example Feature

      Scenario: Scenario A
        Given set state to "1"
        Then state should be "1"

      Scenario: Scenario B
        Then state should be "0"
    """
    And the file "./features/steps/step_test.go" exists with content
    """
    package gorkin

    import (
        "testing"

        . "github.com/kat-co/gorkin/gorkin"
    )

    func Test(t *testing.T) {

        type I struct {
            State int
        }

        Step(`set state to "([^"]+)"$`, func(i *I, state int) {
            i.State = state
        })

        Step(`state should be "([^"]+)"$`, func(i *I, state int) {
            if i.State != state {
                t.Error("current state:", i.State)
                t.Fatal("State is not", state)
            }
        })

        RunFeatureTests(t, &I{})
    }
    """
    When a user runs "gorkin"
    Then the output should contain
    """
    ok
    """
