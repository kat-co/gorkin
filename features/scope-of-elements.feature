Feature: Gorkin Properly Scoping
  In order to allow for repeatable tests
  As a gorkin user
  I want to be able to specify steps in a common way
  And not have the steps from different scenarios affect each other

  Scenario: A User Specifies Conflicting Scenarios
    Given a gorkin feature with 2 scenarios
    And a step which conflicts with another
    When a user runs gorkin
    Then the steps should not conflict with one another
