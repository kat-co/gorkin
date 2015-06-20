Feature: Gorkin help.
  AS A gorkin user
  I WOULD LIKE gorkin --help to provide information about its usage
  SO THAT I can lookup commands.

  Scenario: User runs "gorkin --help"
    When a user runs "gorkin --help"
    Then the output should be
    """
    gorkin v0.0.1:
      -help=false: Get usage on gorkin.
      -init=false: Initialize a Gherkin structure.

    """

    Scenario: A user runs "gorkin --init"
      When a user runs "gorkin --init"
      Then the output should be
      """
      Gherkin structure generated.

      """
