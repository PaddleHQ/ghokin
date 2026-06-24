Feature: A Feature

  Scenario Outline: A scenario with outline vars in JSON
    Given a thing
      """json
      {
        "seller_id": <seller_id>,
        "visibility": "<visibility>",
        "name": "Test"
      }
      """

    Examples:
      | seller_id | visibility |
      | null      | public     |
      | "123"     | private    |
