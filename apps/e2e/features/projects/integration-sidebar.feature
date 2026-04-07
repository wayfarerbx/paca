@ui @sidebar @integrations
Feature: Sidebar integration navigation
  The project sidebar exposes all open integrations — product backlog and
  any sprints that are active or planned — as direct navigation entries
  so that users never need to visit the Integrations settings page just
  to navigate to their work.  Completed sprints are hidden from the sidebar
  by default.  The active integration entry is always highlighted, and the
  section can be collapsed or expanded to manage vertical space.

  @authenticated
  Rule: Displaying open integrations in the project sidebar

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_SIDEBAR_INTEGRATIONS_PROJECT" exists
      And the user has the "View Sprints" project permission in "E2E_SIDEBAR_INTEGRATIONS_PROJECT"
      And the user is inside the project "E2E_SIDEBAR_INTEGRATIONS_PROJECT"

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

    Scenario: Sidebar shows an "Integrations" section label grouping all integration entries
      Then the sidebar should show a section label "Integrations" or "Sprints & Backlog"
      And the entries for product backlog and sprints should appear beneath that label

  @authenticated
  Rule: Navigating directly to an integration from the sidebar

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_NAV_INTEGRATIONS_PROJECT" exists
      And the project has a "Product Backlog" integration
      And the project has an active sprint named "E2E_NAV_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_NAV_INTEGRATIONS_PROJECT"
      And the user is inside the project "E2E_NAV_INTEGRATIONS_PROJECT"

    Scenario: Clicking "Product Backlog" in the sidebar navigates to the backlog integration page
      When the user clicks "Product Backlog" in the project sidebar
      Then the user should be on the "Product Backlog" integration view page
      And the "Product Backlog" sidebar entry should be marked as active

    Scenario: Clicking a sprint in the sidebar navigates to that sprint's integration page
      When the user clicks "E2E_NAV_SPRINT" in the project sidebar
      Then the user should be on the "E2E_NAV_SPRINT" integration view page
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
  Rule: Sidebar integration section collapses and expands

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_COLLAPSE_PROJECT" exists
      And the project has a "Product Backlog" integration and an active sprint "E2E_COLLAPSE_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_COLLAPSE_PROJECT"
      And the user is inside the project "E2E_COLLAPSE_PROJECT"

    Scenario: Clicking the "Integrations" section label collapses all integration entries
      Given the "Integrations" section is expanded
      When the user clicks the "Integrations" section label toggle
      Then the "Product Backlog" entry should not be visible
      And the "E2E_COLLAPSE_SPRINT" entry should not be visible

    Scenario: Clicking the section label again expands all integration entries
      Given the "Integrations" section is collapsed
      When the user clicks the "Integrations" section label toggle
      Then the "Product Backlog" entry should be visible
      And the "E2E_COLLAPSE_SPRINT" entry should be visible

    Scenario: Collapsed state is preserved across page navigation within the project
      Given the "Integrations" section is collapsed
      When the user clicks the "Team" navigation link
      And the user presses the browser back button
      Then the "Integrations" section should still be collapsed

  @authenticated
  Rule: Sidebar hides integration entries without the required permission

    Scenario: User without "View Sprints" permission does not see sprint entries
      Given the user already has a stored authenticated session
      And a project named "E2E_PERM_INTEGRATIONS_PROJECT" exists
      And the project has an active sprint "E2E_HIDDEN_SPRINT"
      And the user does not have the "View Sprints" project permission
      And the user is inside the project "E2E_PERM_INTEGRATIONS_PROJECT"
      Then the project sidebar should not contain an entry for "E2E_HIDDEN_SPRINT"
      And the "Integrations" section label should not be visible

    Scenario: User with "View Sprints" permission sees sprint entries even with no other project permissions
      Given the user already has a stored authenticated session
      And a project named "E2E_PERM_VISIBLE_PROJECT" exists
      And the project has a "Product Backlog" integration
      And the project has an active sprint "E2E_VISIBLE_SPRINT"
      And the user has only the "View Sprints" project permission in "E2E_PERM_VISIBLE_PROJECT"
      And the user is inside the project "E2E_PERM_VISIBLE_PROJECT"
      Then the project sidebar should contain "Product Backlog"
      And the project sidebar should contain "E2E_VISIBLE_SPRINT"

  @authenticated
  Rule: Sidebar integration list updates in real time

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
  Rule: Collapsed global sidebar still shows integration indicators

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_ICON_PROJECT" exists
      And the project has an active sprint "E2E_ICON_SPRINT"
      And the user has the "View Sprints" project permission in "E2E_ICON_PROJECT"
      And the user is inside the project "E2E_ICON_PROJECT"

    Scenario: Collapsed sidebar shows the integration icon for Product Backlog with a tooltip
      Given the global sidebar is in collapsed mode
      When the user hovers over the "Product Backlog" icon in the sidebar
      Then a tooltip labelled "Product Backlog" should be visible

    Scenario: Collapsed sidebar shows the active sprint icon with its name as a tooltip
      Given the global sidebar is in collapsed mode
      When the user hovers over the icon for "E2E_ICON_SPRINT" in the sidebar
      Then a tooltip labelled "E2E_ICON_SPRINT" should be visible
