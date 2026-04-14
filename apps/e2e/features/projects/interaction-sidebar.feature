@ui @sidebar @interactions
Feature: Sidebar interaction navigation
  The project sidebar exposes all open interactions — product backlog and
  any sprints that are active or planned — as direct navigation entries
  so that users never need to visit the Interactions settings page just
  to navigate to their work.  Sprint entries are only visible when the
  Interactions section is in its expanded (open) state; they are hidden
  when the section is collapsed.  Completed sprints are always hidden from
  the sidebar.  The active interaction entry is always highlighted, and the
  section can be collapsed or expanded to manage vertical space.

  @authenticated
  Rule: Displaying open interactions in the project sidebar

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_SIDEBAR_INTERACTIONS_PROJECT" exists
      And the user has the "View Sprints" project permission in "E2E_SIDEBAR_INTERACTIONS_PROJECT"
      And the user is inside the project "E2E_SIDEBAR_INTERACTIONS_PROJECT"

    Scenario: Product Backlog always appears as a fixed entry in the sidebar
      Then the project sidebar should contain a "Product Backlog" entry below the main navigation links

    Scenario: Active sprint appears as an entry in the sidebar
      Given the project has an active sprint named "E2E_ACTIVE_SPRINT"
      Then the project sidebar should contain an entry for "E2E_ACTIVE_SPRINT"

    Scenario: Planned (upcoming) sprint appears as an entry in the sidebar
      Given the project has a planned sprint named "E2E_PLANNED_SPRINT"
      Then the project sidebar should contain an entry for "E2E_PLANNED_SPRINT"

    Scenario: Completed sprint does not appear in the sidebar
      Given the project has a completed sprint named "E2E_COMPLETED_SPRINT"
      Then the project sidebar should not contain an entry for "E2E_COMPLETED_SPRINT"

    Scenario: Multiple open sprints all appear in the sidebar
      Given the project has active sprints "E2E_SPRINT_A" and "E2E_SPRINT_B"
      And the project has a planned sprint "E2E_SPRINT_C"
      Then the sidebar should list entries for "E2E_SPRINT_A", "E2E_SPRINT_B", and "E2E_SPRINT_C"

    Scenario: Sprint entries are ordered with active sprints before planned sprints
      Given the project has a planned sprint "E2E_PLANNED_SPRINT"
      And the project has an active sprint "E2E_ACTIVE_SPRINT"
      Then the "E2E_ACTIVE_SPRINT" entry should appear above the "E2E_PLANNED_SPRINT" entry

    Scenario: Active sprint entry carries a visual indicator for its sprint state
      Given the project has an active sprint named "E2E_RUNNING_SPRINT"
      Then the "E2E_RUNNING_SPRINT" sidebar entry should carry a visual indicator that marks it as active

    Scenario: Sidebar shows an "Interactions" section label grouping all interaction entries
      Then the sidebar should show a section label "Interactions" or "Sprints & Backlog"
      And the entries for product backlog and sprints should appear beneath that label

  @authenticated
  Rule: Navigating directly to an interaction from the sidebar

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_NAV_INTERACTIONS_PROJECT" exists
      And the project has a "Product Backlog" interaction
      And the project has an active sprint named "E2E_NAV_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_NAV_INTERACTIONS_PROJECT"
      And the user is inside the project "E2E_NAV_INTERACTIONS_PROJECT"

    Scenario: Clicking "Product Backlog" in the sidebar navigates to the backlog interaction page
      When the user clicks "Product Backlog" in the project sidebar
      Then the user should be on the "Product Backlog" interaction view page
      And the "Product Backlog" sidebar entry should be marked as active

    Scenario: Clicking a sprint in the sidebar navigates to that sprint's interaction page
      When the user clicks "E2E_NAV_SPRINT" in the project sidebar
      Then the user should be on the "E2E_NAV_SPRINT" interaction view page
      And the "E2E_NAV_SPRINT" sidebar entry should be marked as active

    Scenario: Navigating to a sprint deactivates the previously active entry
      Given the user has navigated to "Product Backlog" from the sidebar
      When the user clicks "E2E_NAV_SPRINT" in the project sidebar
      Then the "E2E_NAV_SPRINT" entry should be marked as active
      And the "Product Backlog" entry should no longer be marked as active

    Scenario: The sidebar active state persists after a page refresh
      Given the user has navigated to the "E2E_NAV_SPRINT" page
      When the user refreshes the browser
      Then the "E2E_NAV_SPRINT" sidebar entry should still be marked as active

  @authenticated
  Rule: Sidebar interaction section collapses and expands

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_COLLAPSE_PROJECT" exists
      And the project has a "Product Backlog" interaction and an active sprint "E2E_COLLAPSE_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_COLLAPSE_PROJECT"
      And the user is inside the project "E2E_COLLAPSE_PROJECT"

    Scenario: Clicking the "Interactions" section label collapses all interaction entries
      Given the "Interactions" section is expanded
      When the user clicks the "Interactions" section label toggle
      Then the "Product Backlog" entry should not be visible
      And the "E2E_COLLAPSE_SPRINT" entry should not be visible

    Scenario: Clicking the section label again expands all interaction entries
      Given the "Interactions" section is collapsed
      When the user clicks the "Interactions" section label toggle
      Then the "Product Backlog" entry should be visible
      And the "E2E_COLLAPSE_SPRINT" entry should be visible

    Scenario: Collapsed state is preserved across page navigation within the project
      Given the "Interactions" section is collapsed
      When the user clicks the "Team" navigation link
      And the user presses the browser back button
      Then the "Interactions" section should still be collapsed

  @authenticated
  Rule: Sidebar hides interaction entries without the required permission

    Scenario: User without "View Sprints" permission does not see sprint entries
      Given the user already has a stored authenticated session
      And a project named "E2E_PERM_INTERACTIONS_PROJECT" exists
      And the project has an active sprint "E2E_HIDDEN_SPRINT"
      And the user does not have the "View Sprints" project permission
      And the user is inside the project "E2E_PERM_INTERACTIONS_PROJECT"
      Then the project sidebar should not contain an entry for "E2E_HIDDEN_SPRINT"
      And the "Interactions" section label should not be visible

    Scenario: User with "View Sprints" permission sees sprint entries even with no other project permissions
      Given the user already has a stored authenticated session
      And a project named "E2E_PERM_VISIBLE_PROJECT" exists
      And the project has a "Product Backlog" interaction
      And the project has an active sprint "E2E_VISIBLE_SPRINT"
      And the user has only the "View Sprints" project permission in "E2E_PERM_VISIBLE_PROJECT"
      And the user is inside the project "E2E_PERM_VISIBLE_PROJECT"
      Then the project sidebar should contain "Product Backlog"
      And the project sidebar should contain "E2E_VISIBLE_SPRINT"

  @authenticated
  Rule: Sidebar interaction list updates in real time

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_REALTIME_PROJECT" exists
      And the user has the "View Sprints" project permission in "E2E_REALTIME_PROJECT"
      And the user is inside the project "E2E_REALTIME_PROJECT"

    Scenario: Starting a sprint adds it to the sidebar without a full page reload
      Given the project currently has no active sprints
      When another user or admin starts a new sprint "E2E_LIVE_SPRINT" in the project
      Then the sidebar entry for "E2E_LIVE_SPRINT" should appear without the user reloading the page

    Scenario: Completing a sprint removes it from the sidebar without a full page reload
      Given the sidebar shows an active sprint "E2E_COMPLETING_SPRINT"
      When another user or admin completes the sprint "E2E_COMPLETING_SPRINT"
      Then the sidebar entry for "E2E_COMPLETING_SPRINT" should disappear without the user reloading the page

  @authenticated
  Rule: Collapsed global sidebar still shows interaction indicators

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_ICON_PROJECT" exists
      And the project has an active sprint "E2E_ICON_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_ICON_PROJECT"
      And the user is inside the project "E2E_ICON_PROJECT"

    Scenario: Collapsed sidebar shows the interaction icon for Product Backlog with a tooltip
      Given the global sidebar is in collapsed mode
      When the user hovers over the "Product Backlog" icon in the sidebar
      Then a tooltip labelled "Product Backlog" should be visible

    Scenario: Collapsed sidebar shows the active sprint icon with its name as a tooltip
      Given the global sidebar is in collapsed mode
      When the user hovers over the icon for "E2E_ICON_SPRINT" in the sidebar
      Then a tooltip labelled "E2E_ICON_SPRINT" should be visible

  @authenticated
  Rule: Sprint entries are only visible when the Interactions section is expanded

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_SPRINT_VISIBILITY_PROJECT" exists
      And the project has an active sprint named "E2E_VISIBILITY_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_SPRINT_VISIBILITY_PROJECT"
      And the user is inside the project "E2E_SPRINT_VISIBILITY_PROJECT"

    Scenario: Sprint entries are visible when the Interactions section is expanded
      Given the "Interactions" section is expanded
      Then the project sidebar should contain an entry for "E2E_VISIBILITY_SPRINT"

    Scenario: Sprint entries are hidden when the Interactions section is collapsed
      Given the "Interactions" section is expanded
      When the user clicks the "Interactions" section label toggle to collapse it
      Then the project sidebar should not contain a visible entry for "E2E_VISIBILITY_SPRINT"

    Scenario: Expanding the section again reveals sprint entries
      Given the "Interactions" section is collapsed
      When the user clicks the "Interactions" section label toggle to expand it
      Then the project sidebar should contain an entry for "E2E_VISIBILITY_SPRINT"

  @authenticated
  Rule: Dragging a task onto a sidebar interaction entry reassigns its sprint

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_DRAG_SPRINT_PROJECT" exists
      And the project has a "Product Backlog" interaction
      And the project has an active sprint named "E2E_TARGET_SPRINT"
      And the project has an active sprint named "E2E_SOURCE_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_DRAG_SPRINT_PROJECT"
      And the user has the "Edit Tasks" project permission in "E2E_DRAG_SPRINT_PROJECT"
      And the user has navigated to the "E2E_SOURCE_SPRINT" board view inside "E2E_DRAG_SPRINT_PROJECT"

    Scenario: Dragging a task card onto a sprint sidebar entry moves the task into that sprint
      Given the interaction has a task "E2E_MOVE_TASK" in sprint "E2E_SOURCE_SPRINT"
      When the user drags the task card "E2E_MOVE_TASK" onto the "E2E_TARGET_SPRINT" sidebar entry
      Then the task "E2E_MOVE_TASK" should no longer appear in the "E2E_SOURCE_SPRINT" view
      And the task "E2E_MOVE_TASK" should appear in the "E2E_TARGET_SPRINT" view

    Scenario: Dragging a task row from the table view onto a sprint sidebar entry moves the task
      Given the user has navigated to the "E2E_SOURCE_SPRINT" table view inside "E2E_DRAG_SPRINT_PROJECT"
      And the interaction has a task "E2E_TABLE_MOVE_TASK" in sprint "E2E_SOURCE_SPRINT"
      When the user drags the task row "E2E_TABLE_MOVE_TASK" onto the "E2E_TARGET_SPRINT" sidebar entry
      Then the task "E2E_TABLE_MOVE_TASK" should appear in the "E2E_TARGET_SPRINT" view

    Scenario: Dragging a task onto the "Product Backlog" sidebar entry removes it from its sprint
      Given the interaction has a task "E2E_BACKLOG_TASK" in sprint "E2E_SOURCE_SPRINT"
      When the user drags the task card "E2E_BACKLOG_TASK" onto the "Product Backlog" sidebar entry
      Then the task "E2E_BACKLOG_TASK" should no longer appear in the "E2E_SOURCE_SPRINT" view
      And the task "E2E_BACKLOG_TASK" should appear in the "Product Backlog" view

    Scenario: The sprint sidebar entry highlights as a drop target when a task is dragged over it
      Given the interaction has a task "E2E_HOVER_TASK" in sprint "E2E_SOURCE_SPRINT"
      When the user starts dragging the task card "E2E_HOVER_TASK"
      And the user hovers the dragged task over the "E2E_TARGET_SPRINT" sidebar entry
      Then the "E2E_TARGET_SPRINT" sidebar entry should display a visual drop-target highlight

    Scenario: The drop-target highlight is removed when the task is dragged away from the entry
      Given the interaction has a task "E2E_LEAVE_TASK" in sprint "E2E_SOURCE_SPRINT"
      When the user starts dragging the task card "E2E_LEAVE_TASK"
      And the user hovers the dragged task over the "E2E_TARGET_SPRINT" sidebar entry
      And the user moves the dragged task away from the "E2E_TARGET_SPRINT" sidebar entry
      Then the "E2E_TARGET_SPRINT" sidebar entry should no longer show the drop-target highlight

    Scenario: User without "Edit Tasks" permission cannot reassign a task via sidebar drag
      Given the user does not have the "Edit Tasks" project permission
      And the interaction has a task "E2E_READONLY_DRAG_TASK" in sprint "E2E_SOURCE_SPRINT"
      When the user attempts to drag "E2E_READONLY_DRAG_TASK" onto the "E2E_TARGET_SPRINT" sidebar entry
      Then the drag operation should not be accepted by the sidebar entry
      And the task "E2E_READONLY_DRAG_TASK" should remain in "E2E_SOURCE_SPRINT"
