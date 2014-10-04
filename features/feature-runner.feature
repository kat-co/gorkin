Feature: Running Over Feature Files
  In order to consume feature files
  A gorkin user will need to run gorkin against a directory of files.

  Background:
    Given there is a directory named "features"
    And there is at least 1 feature file

  Scenario: A user runs gorkin
    When the user runs gorkin
    Then gorkin should find the features directory
    And it should find the .feature files within that directory

  Scenario: A user runs gorkin without a features directory
    Given there is not a directory named "features"
    When the user runs gorkin
    Then they should receive an error
