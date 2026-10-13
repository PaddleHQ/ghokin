# language: en
Feature: Comments and tags inside rules

# above first rule
@ruletag
  Rule: First

  # above background
    Background:
      Given a background step

  # above scenario
  @tag
    Scenario: One
    # above step
      Given a thing
    # between steps
      Then b

  # above outline
    Scenario Outline: Two
      Given <x>

    # above examples
    @examplestag
      Examples:
        | x |
        | 1 |

# above second rule
@ruletag
  Rule: Second

    Scenario: Three
      Given c
  # trailing
