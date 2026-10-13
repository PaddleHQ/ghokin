Feature: JSON docstrings inside rules

  Scenario: Outside any rule
    When I make a POST call to /x with body
      """json
      {"a": "b", "c": [1, 2]}
      """

  Rule: First

    Scenario: Thing - 200
      When I make a POST call to /x with body
      """json
      {"a": "b",
      "c": [1, 2]}
        """

    Scenario Outline: Outline - <status>
      When I make a POST call to /x with body
          """json
          <body>
          """
      Then the response status is <status>

      Examples:
        | body       | status |
        | {"a": "b"} | 200    |
        | {}         | 400    |
