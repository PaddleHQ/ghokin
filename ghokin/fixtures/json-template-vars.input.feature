Feature: A Feature

  Scenario: A scenario with template vars in JSON
    Given a thing
      """json
      {"name": "{{ name }}", "count": {{ count }},
      "items": [{"id": "{{ item_id }}"}]}
      """
