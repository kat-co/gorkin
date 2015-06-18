Feature: Running Over Feature Files
  In order to consume feature files
  A gorkin user will need to run gorkin against a directory of files.

  Scenario: A user runs gorkin without a features directory
    Given there is not a directory named "features"
    When a user runs gorkin
    Then they should receive this error
    """
    could not find a features directory.
    """

  Scenario: A user runs gorkin
    When there is a directory named "features"
    And there is at least 1 feature file
    And there is a steps directory under features
    And there is at least 1 gorkin test file
    And a user runs gorkin
    Then gorkin should find the features directory
    And it should find the .feature files within that directory
