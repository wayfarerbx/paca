@projects @sprints @sprint-lifecycle
Feature: Sprint lifecycle management
  Sprints move through three states: planned → active → completed.  A sprint
  is created in the planned state with a system-generated default name
  ("Sprint N") and no dates.  Authorised users start a planned sprint by
  clicking the "Start sprint" button — either from the sprint column header on
  the product backlog Table view or from the sprint interaction page header —
  which opens the Start Sprint modal where they confirm (or edit) the name,
  goal, start date, and due date before activating it.  Multiple sprints may
  be active at the same time.  Authorised users complete an active sprint
  by clicking the "Complete sprint" button on the sprint interaction page,
  which opens the Complete Sprint modal prompting them to choose a destination
  sprint for any remaining incomplete tasks.  Sprints in the planned state can
  also be deleted.  Sprint creation is always a quick-create action triggered
  from the product backlog page header or sprint column headers — no creation
  modal is shown.

  @authenticated
  Rule: Creating a sprint (quick create — no modal)

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_QUICK_CREATE_PROJECT" exists
      And the user has the "View Sprints" project permission in "E2E_QUICK_CREATE_PROJECT"
      And the user has the "Manage Sprints" project permission in "E2E_QUICK_CREATE_PROJECT"
      And the user has navigated to the "Product Backlog" table view inside "E2E_QUICK_CREATE_PROJECT"

    Scenario: Clicking "New sprint" in the page header creates a sprint with a default name
      When the user clicks "New sprint" in the product backlog page header
      Then a new sprint should appear in the sidebar within the planned state
      And the sprint name should match the pattern "Sprint \d+"
      And no creation modal or dialog should appear

    Scenario: Sequential quick-creates produce incrementally numbered names
      Given the project has no existing sprints
      When the user clicks "New sprint" in the product backlog page header
      Then the first sprint should be named "Sprint 1"
      When the user clicks "New sprint" in the product backlog page header again
      Then the second sprint should be named "Sprint 2"

    Scenario: A quick-created sprint appears as a column in the product backlog table view
      When the user clicks "New sprint" in the product backlog page header
      Then a new sprint column should appear in the product backlog table view

    Scenario: A quick-created sprint has "planned" status
      When the user clicks "New sprint" in the product backlog page header
      Then the new sprint should have status "planned"

    Scenario: "New sprint" button is not visible to users without "Manage Sprints" permission
      Given the user does not have the "Manage Sprints" project permission
      Then the "New sprint" button should not be visible in the product backlog page header

  @authenticated
  Rule: Starting a sprint

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_START_SPRINT_PROJECT" exists
      And the project has a planned sprint named "E2E_START_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_START_SPRINT_PROJECT"
      And the user has the "Manage Sprints" project permission in "E2E_START_SPRINT_PROJECT"
      And the user has navigated to the "Product Backlog" table view inside "E2E_START_SPRINT_PROJECT"

    Scenario: "Start sprint" button appears in the header of a planned sprint column
      Then the column header for "E2E_START_SPRINT" should contain a "Start sprint" button

    Scenario: Clicking "Start sprint" opens the Start Sprint modal
      When the user clicks "Start sprint" in the "E2E_START_SPRINT" column header
      Then the "Start sprint" modal should open

    Scenario: Start Sprint modal shows sprint name, goal, start date, and due date fields
      When the user clicks "Start sprint" in the "E2E_START_SPRINT" column header
      Then the modal should display a pre-filled "Sprint name" field containing "E2E_START_SPRINT"
      And the modal should contain an optional "Goal" text field
      And the modal should contain an optional "Start date" date picker
      And the modal should contain an optional "Due date" date picker

    Scenario: Submitting the modal with only the default name starts the sprint
      When the user clicks "Start sprint" in the "E2E_START_SPRINT" column header
      And the user clicks "Start sprint" in the modal without changing any fields
      Then the modal should close
      And the sprint "E2E_START_SPRINT" should have status "active"

    Scenario: Submitting the modal after setting goal and dates saves all values
      When the user clicks "Start sprint" in the "E2E_START_SPRINT" column header
      And the user fills the goal with "Deliver authentication"
      And the user sets the start date to "2026-04-14"
      And the user sets the due date to "2026-04-27"
      And the user clicks "Start sprint" in the modal
      Then the sprint "E2E_START_SPRINT" should have status "active"
      And the sprint "E2E_START_SPRINT" should have goal "Deliver authentication"
      And the sprint "E2E_START_SPRINT" should have start date "2026-04-14" and due date "2026-04-27"

    Scenario: Renaming the sprint in the Start Sprint modal updates the sprint name
      When the user clicks "Start sprint" in the "E2E_START_SPRINT" column header
      And the user clears the name field and types "E2E_RENAMED_SPRINT"
      And the user clicks "Start sprint" in the modal
      Then the sprint formerly named "E2E_START_SPRINT" should now be named "E2E_RENAMED_SPRINT"
      And it should have status "active"

    Scenario: Cancelling the modal leaves the sprint in planned state
      When the user clicks "Start sprint" in the "E2E_START_SPRINT" column header
      And the user fills the goal with "Should not be saved"
      And the user clicks "Cancel" in the modal
      Then the modal should close
      And the sprint "E2E_START_SPRINT" should still have status "planned"

    Scenario: "Start sprint" button is not shown on an active sprint column
      Given the project has an active sprint named "E2E_ACTIVE_SPRINT_COL"
      And the user has navigated to the "Product Backlog" table view inside "E2E_START_SPRINT_PROJECT"
      Then the column header for "E2E_ACTIVE_SPRINT_COL" should not contain a "Start sprint" button

  @authenticated
  Rule: Completing a sprint

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_COMPLETE_SPRINT_PROJECT" exists
      And the project has an active sprint named "E2E_COMPLETE_SPRINT"
      And the project has a planned sprint named "E2E_NEXT_SPRINT"
      And the sprint "E2E_COMPLETE_SPRINT" has incomplete tasks "E2E_INCOMPLETE_TASK_1" and "E2E_INCOMPLETE_TASK_2"
      And the sprint "E2E_COMPLETE_SPRINT" has a completed task "E2E_DONE_TASK" with status category "done"
      And the user has the "View Sprints" project permission in "E2E_COMPLETE_SPRINT_PROJECT"
      And the user has the "Manage Sprints" project permission in "E2E_COMPLETE_SPRINT_PROJECT"
      And the user has navigated to the "E2E_COMPLETE_SPRINT" sprint page inside "E2E_COMPLETE_SPRINT_PROJECT"

    Scenario: A "Complete sprint" button is visible on the sprint interaction page header
      Then the sprint page header should contain a "Complete sprint" button

    Scenario: Clicking "Complete sprint" opens the Complete Sprint modal
      When the user clicks "Complete sprint" in the sprint page header
      Then the "Complete sprint" modal should open

    Scenario: Complete Sprint modal shows the sprint name and incomplete task count
      When the user clicks "Complete sprint" in the sprint page header
      Then the modal should display the sprint name "E2E_COMPLETE_SPRINT"
      And the modal should indicate that 2 tasks are incomplete

    Scenario: Complete Sprint modal offers a dropdown to select a destination sprint for incomplete tasks
      When the user clicks "Complete sprint" in the sprint page header
      Then the modal should contain a sprint selector for incomplete tasks
      And the selector should list "E2E_NEXT_SPRINT" as an option
      And the selector should include a "Product Backlog (no sprint)" option

    Scenario: Confirming completion with a destination sprint moves incomplete tasks there
      When the user clicks "Complete sprint" in the sprint page header
      And the user selects "E2E_NEXT_SPRINT" as the destination for incomplete tasks
      And the user clicks "Complete sprint" in the modal
      Then the modal should close
      And the sprint "E2E_COMPLETE_SPRINT" should have status "completed"
      And "E2E_INCOMPLETE_TASK_1" should now be assigned to sprint "E2E_NEXT_SPRINT"
      And "E2E_INCOMPLETE_TASK_2" should now be assigned to sprint "E2E_NEXT_SPRINT"

    Scenario: Confirming completion with "Product Backlog" moves incomplete tasks to no sprint
      When the user clicks "Complete sprint" in the sprint page header
      And the user selects "Product Backlog (no sprint)" as the destination for incomplete tasks
      And the user clicks "Complete sprint" in the modal
      Then the modal should close
      And the sprint "E2E_COMPLETE_SPRINT" should have status "completed"
      And "E2E_INCOMPLETE_TASK_1" should have no sprint assigned
      And "E2E_INCOMPLETE_TASK_2" should have no sprint assigned

    Scenario: Completed tasks remain on the completed sprint after closing
      When the user clicks "Complete sprint" in the sprint page header
      And the user selects "Product Backlog (no sprint)" as the destination
      And the user clicks "Complete sprint" in the modal
      Then the task "E2E_DONE_TASK" should still be associated with sprint "E2E_COMPLETE_SPRINT"

    Scenario: Cancelling the Complete Sprint modal leaves the sprint active
      When the user clicks "Complete sprint" in the sprint page header
      And the user clicks "Cancel" in the complete sprint modal
      Then the modal should close
      And the sprint "E2E_COMPLETE_SPRINT" should still have status "active"
      And "E2E_INCOMPLETE_TASK_1" should still be in sprint "E2E_COMPLETE_SPRINT"

    Scenario: Completed sprints do not appear in the project sidebar
      When the sprint "E2E_COMPLETE_SPRINT" is completed
      Then the project sidebar should not contain an entry for "E2E_COMPLETE_SPRINT"

    Scenario: "Complete sprint" button is not visible without "Manage Sprints" permission
      Given the user does not have the "Manage Sprints" project permission
      When the user navigates to the "E2E_COMPLETE_SPRINT" sprint page
      Then the sprint page header should not contain a "Complete sprint" button

  @authenticated
  Rule: Sprint lifecycle state constraints

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_LIFECYCLE_CONSTRAINTS_PROJECT" exists
      And the user has the "View Sprints" project permission in "E2E_LIFECYCLE_CONSTRAINTS_PROJECT"
      And the user has the "Manage Sprints" project permission in "E2E_LIFECYCLE_CONSTRAINTS_PROJECT"

    Scenario: Multiple sprints can be active at the same time
      Given the project has an active sprint named "E2E_ACTIVE_SPRINT_A"
      And the project has a planned sprint named "E2E_PLANNED_SPRINT_B"
      When the user starts "E2E_PLANNED_SPRINT_B"
      Then both "E2E_ACTIVE_SPRINT_A" and "E2E_PLANNED_SPRINT_B" should have status "active"

    Scenario: A planned sprint can be deleted
      Given the project has a planned sprint named "E2E_DELETE_PLANNED_SPRINT"
      When the user deletes the sprint "E2E_DELETE_PLANNED_SPRINT"
      Then the sprint should be removed from the project

    Scenario: An active sprint cannot be deleted
      Given the project has an active sprint named "E2E_ACTIVE_NO_DELETE"
      When the user attempts to delete the sprint "E2E_ACTIVE_NO_DELETE"
      Then an error should indicate that active sprints cannot be deleted

    Scenario: A completed sprint is read-only and cannot be restarted
      Given the project has a completed sprint named "E2E_COMPLETED_SPRINT"
      When the user navigates to the "E2E_COMPLETED_SPRINT" sprint page
      Then the sprint page header should not contain a "Start sprint" button
      And the sprint page header should not contain a "Complete sprint" button
