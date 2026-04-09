@projects @custom-fields
Feature: Custom field management
  Project members with the appropriate permissions can define custom fields
  that extend the default task properties for their project.  Custom fields
  are managed from Settings > Custom Fields on the project settings page.
  Each field has a required display name, an auto-generated (but editable)
  field key derived from the display name, a field type chosen from Text,
  Number, Date, Checkbox, or Select, an options list (Select type only), and
  an optional Required toggle.  New projects start with no custom fields.
  The field key is immutable after creation and the field type cannot be
  changed once the field has been saved.

  @authenticated
  Rule: Viewing custom fields

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_CFIELD_VIEW_PROJECT" exists
      And the user has navigated to the "E2E_CFIELD_VIEW_PROJECT" Settings page

    Scenario: Custom Fields section is reachable from the settings sidebar
      When the user clicks "Custom Fields" in the settings sidebar
      Then the "Custom Fields" section heading should be visible
      And the section should display a description mentioning extending tasks with custom fields

    Scenario: New project shows an empty state with no custom fields
      When the user clicks "Custom Fields" in the settings sidebar
      Then the custom fields section should show an empty state
      And the empty state should invite the user to create their first custom field

    Scenario: Custom fields table shows the expected columns once fields exist
      Given a custom field named "E2E Priority Score" exists in the project
      When the user clicks "Custom Fields" in the settings sidebar
      Then the custom fields table should have columns "Display Name", "Field Key", "Type", and "Required"

    Scenario: Each custom field row has Edit and Delete action buttons
      Given a custom field named "E2E Priority Score" exists in the project
      When the user clicks "Custom Fields" in the settings sidebar
      Then every custom field row should have an "Edit field" button
      And every custom field row should have a "Delete field" button

    Scenario: "New custom field" button is visible on the Custom Fields section
      When the user clicks "Custom Fields" in the settings sidebar
      Then the "New custom field" button should be visible

    Scenario: Field key is displayed in monospace style
      Given a custom field named "E2E Priority Score" exists in the project
      When the user clicks "Custom Fields" in the settings sidebar
      Then the field key cell for "E2E Priority Score" should be rendered in monospace style

  @authenticated
  Rule: Creating a custom field

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_CFIELD_CREATE_PROJECT" exists
      And the user has navigated to the "E2E_CFIELD_CREATE_PROJECT" Settings page
      And the user clicks "Custom Fields" in the settings sidebar

    Scenario: Opening the create-custom-field dialog
      When the user clicks the "New custom field" button
      Then the "Create custom field" dialog should open
      And the dialog should contain a required "Display name" field
      And the dialog should contain a "Field key" field
      And the dialog should contain a "Field type" selector
      And the dialog should contain a "Required" toggle

    Scenario: "Create field" button is disabled while the display name is empty
      When the user clicks the "New custom field" button
      Then the "Create field" button should be disabled

    Scenario: "Create field" button becomes enabled after typing a display name
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Sprint Points"
      Then the "Create field" button should be enabled

    Scenario: Typing a display name auto-generates the field key
      When the user clicks the "New custom field" button
      And the user fills the display name with "Sprint Points"
      Then the "Field key" field should be auto-populated with "sprint_points"

    Scenario: Auto-generated field key can be manually overridden
      When the user clicks the "New custom field" button
      And the user fills the display name with "Sprint Points"
      And the user clears the field key and types "sp_pts"
      Then the "Field key" field should contain "sp_pts"

    Scenario: Field type selector lists all available types
      When the user clicks the "New custom field" button
      Then the field type selector should offer "Text", "Number", "Date", "Checkbox", and "Select"

    Scenario: Default field type is Text
      When the user clicks the "New custom field" button
      Then the "Text" type should be selected by default

    Scenario: Options editor is hidden for non-Select field types
      When the user clicks the "New custom field" button
      And the user selects field type "Number"
      Then the options editor should not be visible

    Scenario: Options editor appears when the Select field type is chosen
      When the user clicks the "New custom field" button
      And the user selects field type "Select"
      Then the options editor should be visible

    Scenario: Adding an option to a Select field
      When the user clicks the "New custom field" button
      And the user selects field type "Select"
      And the user adds an option "High"
      Then the options list should contain "High"

    Scenario: Adding multiple options to a Select field
      When the user clicks the "New custom field" button
      And the user selects field type "Select"
      And the user adds an option "Low"
      And the user adds an option "Medium"
      And the user adds an option "High"
      Then the options list should contain "Low", "Medium", and "High"

    Scenario: Removing an option from a Select field
      When the user clicks the "New custom field" button
      And the user selects field type "Select"
      And the user adds an option "Unwanted"
      And the user removes the option "Unwanted"
      Then the options list should not contain "Unwanted"

    Scenario: Creating a Text field succeeds
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Notes"
      And the user selects field type "Text"
      And the user clicks "Create field"
      Then the dialog should close
      And the custom fields table should contain a field named "E2E Notes" with type "Text"

    Scenario: Creating a Number field succeeds
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Story Points"
      And the user selects field type "Number"
      And the user clicks "Create field"
      Then the dialog should close
      And the custom fields table should contain a field named "E2E Story Points" with type "Number"

    Scenario: Creating a Date field succeeds
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Due Date"
      And the user selects field type "Date"
      And the user clicks "Create field"
      Then the dialog should close
      And the custom fields table should contain a field named "E2E Due Date" with type "Date"

    Scenario: Creating a Checkbox field succeeds
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Blocked"
      And the user selects field type "Checkbox"
      And the user clicks "Create field"
      Then the dialog should close
      And the custom fields table should contain a field named "E2E Blocked" with type "Checkbox"

    Scenario: Creating a Select field with options succeeds
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Severity"
      And the user selects field type "Select"
      And the user adds an option "Low"
      And the user adds an option "Medium"
      And the user adds an option "High"
      And the user clicks "Create field"
      Then the dialog should close
      And the custom fields table should contain a field named "E2E Severity" with type "Select"

    Scenario: Creating a Required field shows "Yes" in the Required column
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Required Field"
      And the user enables the "Required" toggle
      And the user clicks "Create field"
      Then the dialog should close
      And the row for "E2E Required Field" should display "Yes" in the Required column

    Scenario: Creating an optional field shows "No" in the Required column
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Optional Field"
      And the user clicks "Create field"
      Then the dialog should close
      And the row for "E2E Optional Field" should display "No" in the Required column

    Scenario: Created field key matches the auto-generated slug
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Sprint Goal"
      And the user clicks "Create field"
      Then the dialog should close
      And the row for "E2E Sprint Goal" should display the field key "e2e_sprint_goal"

    Scenario: Cancelling the create dialog discards changes
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Should Not Exist"
      And the user clicks "Cancel"
      Then the dialog should close
      And the custom fields table should not contain a field named "E2E Should Not Exist"

    Scenario: Closing the create dialog with the Close button discards changes
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Should Not Exist via X"
      And the user clicks the Close button on the dialog
      Then the dialog should close
      And the custom fields table should not contain a field named "E2E Should Not Exist via X"

    Scenario: Duplicate field key within the same project is rejected
      Given a custom field with field key "e2e_dup_key" already exists in the project
      When the user clicks the "New custom field" button
      And the user fills the display name with "E2E Duplicate"
      And the user clears the field key and types "e2e_dup_key"
      And the user clicks "Create field"
      Then the dialog should remain open
      And an error should indicate that the field key is already in use

  @authenticated
  Rule: Editing a custom field

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_CFIELD_EDIT_PROJECT" exists
      And a Select custom field named "E2E Edit Me Field" with options "Alpha" and "Beta" exists in the project
      And the user has navigated to the "E2E_CFIELD_EDIT_PROJECT" Settings page
      And the user clicks "Custom Fields" in the settings sidebar

    Scenario: Opening the edit dialog pre-fills existing values
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      Then the "Edit custom field" dialog should open
      And the "Display name" field should be pre-filled with "E2E Edit Me Field"
      And the "Field key" field should be pre-filled and disabled

    Scenario: Field type is not editable in the edit dialog
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      Then the field type selector should be disabled in the edit dialog

    Scenario: Field key is immutable in the edit dialog
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      Then the "Field key" field should be disabled

    Scenario: Saving a new display name updates the field in the table
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      And the user clears the display name and types "E2E Renamed Field"
      And the user clicks "Save changes"
      Then the dialog should close
      And the custom fields table should contain a field named "E2E Renamed Field"
      And the custom fields table should not contain a field named "E2E Edit Me Field"

    Scenario: Clearing the display name disables the save button
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      And the user clears the display name field
      Then the "Save changes" button should be disabled

    Scenario: Options editor is visible for a Select field in edit mode
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      Then the options editor should be visible
      And the options list should contain "Alpha"
      And the options list should contain "Beta"

    Scenario: Adding an option to an existing Select field saves correctly
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      And the user adds an option "Gamma"
      And the user clicks "Save changes"
      Then the dialog should close
      And the field "E2E Edit Me Field" should have the option "Gamma"

    Scenario: Removing an option from an existing Select field saves correctly
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      And the user removes the option "Alpha"
      And the user clicks "Save changes"
      Then the dialog should close
      And the field "E2E Edit Me Field" should no longer have the option "Alpha"

    Scenario: Toggling the Required flag updates the Required column
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      And the user enables the "Required" toggle
      And the user clicks "Save changes"
      Then the dialog should close
      And the row for "E2E Edit Me Field" should display "Yes" in the Required column

    Scenario: Cancelling the edit dialog discards changes
      When the user clicks "Edit field" for the field named "E2E Edit Me Field"
      And the user clears the display name and types "E2E Should Not Save"
      And the user clicks "Cancel"
      Then the dialog should close
      And the custom fields table should still contain a field named "E2E Edit Me Field"
      And the custom fields table should not contain a field named "E2E Should Not Save"

  @authenticated
  Rule: Deleting a custom field

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_CFIELD_DELETE_PROJECT" exists
      And a custom field named "E2E Delete Me Field" exists in the project
      And the user has navigated to the "E2E_CFIELD_DELETE_PROJECT" Settings page
      And the user clicks "Custom Fields" in the settings sidebar

    Scenario: Opening the delete dialog shows a confirmation message
      When the user clicks "Delete field" for the field named "E2E Delete Me Field"
      Then the "Delete custom field" dialog should open
      And the dialog should identify the field being deleted by name
      And the dialog should warn that task data stored in this field will be lost
      And the dialog should warn that the action cannot be undone

    Scenario: Confirming deletion removes the field from the table
      When the user clicks "Delete field" for the field named "E2E Delete Me Field"
      And the user confirms by clicking "Delete field" in the dialog
      Then the dialog should close
      And the custom fields table should not contain a field named "E2E Delete Me Field"

    Scenario: Confirming deletion of the last field returns to the empty state
      Given "E2E Delete Me Field" is the only custom field in the project
      When the user clicks "Delete field" for the field named "E2E Delete Me Field"
      And the user confirms by clicking "Delete field" in the dialog
      Then the custom fields section should show an empty state

    Scenario: Cancelling the delete dialog keeps the field in the table
      When the user clicks "Delete field" for the field named "E2E Delete Me Field"
      And the user clicks "Cancel" in the delete confirmation dialog
      Then the dialog should close
      And the custom fields table should still contain a field named "E2E Delete Me Field"

    Scenario: Closing the delete dialog with the Close button keeps the field
      When the user clicks "Delete field" for the field named "E2E Delete Me Field"
      And the user clicks the Close button on the dialog
      Then the custom fields table should still contain a field named "E2E Delete Me Field"
