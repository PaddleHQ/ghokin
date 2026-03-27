Feature: A Feature

  Scenario: A scenario with embedded template vars in JSON strings
    Given a thing
      """json
      {
        "url": "https://example.com/path?token={{token}}",
        "name": "{{ name }}",
        "count": {{ count }},
        "multi": "a={{a}}&b={{b}}"
      }
      """
