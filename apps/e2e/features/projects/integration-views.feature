@projects @integrations @views
Feature: Integration views (board and list layouts)
  Each project integration (product backlog or a sprint) can be explored
  through one or more saved views.  A view defines a named layout — Board
  or List — and persists the user's preferred way of looking at work items.
  The default view opened when entering an integration is always the first
  view in the tab bar.  Users with sufficient permissions may create additional
  views, rename or delete them, and switch freely between layouts.  Creating
  tasks is available inline from both Board and List views.

  @authenticated
  Rule: Entering an integration opens its default view

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_VIEWS_PROJECT" exists
      And the project has a "Product Backlog" integration with at least one view
      And the user has the "View Sprints" project permission in "E2E_VIEWS_PROJECT"
      And the user has navigated to the "E2E_VIEWS_PROJECT" project

    Scenario: Navigating to the product backlog opens the default view
      When the user clicks "Product Backlog" in the project sidebar
      Then the integration page for "Product Backlog" should be visible
      And a view tab bar should be shown below the integration header
      And the first view tab should be active

    Scenario: The view header shows the integration name and a description
      When the user clicks "Product Backlog" in the project sidebar
      Then the page header should display "Product Backlog"
      And a subtitle or description area should be visible beneath the header

    Scenario: Board view is indicated by its tab label
      Given the "Product Backlog" integration has a view named "Board" with layout "Board"
      When the user clicks "Product Backlog" in the project sidebar
      And the user clicks the "Board" view tab
      Then the kanban board layout should be displayed
      And each status column should be visible as a swimlane

    Scenario: List view is indicated by its tab label
      Given the "Product Backlog" integration has a view named "List" with layout "List"
      When the user clicks "Product Backlog" in the project sidebar
      And the user clicks the "List" view tab
      Then the tabular list layout should be displayed
      And each task should appear as a row with its title, status, type, and assignee

    Scenario: Navigating to a sprint opens that sprint's default view
      Given the project has an active sprint named "E2E_SPRINT_1"
      When the user clicks "E2E_SPRINT_1" in the project sidebar
      Then the integration page for "E2E_SPRINT_1" should be visible
      And the view tab bar should be shown

  @authenticated
  Rule: Board view layout and task display

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_BOARD_PROJECT" exists
      And the project has a "Product Backlog" integration
      And the integration has a "Board" view
      And the user has the "View Sprints" project permission in "E2E_BOARD_PROJECT"
      And the user has navigated to the "Product Backlog" board view inside "E2E_BOARD_PROJECT"

    Scenario: Board columns match the project's configured task statuses
      Then a column should exist for each configured task status
      And the column headers should display the status name and task count

    Scenario: Tasks appear in the column matching their current status
      Given the integration has a task "E2E_TASK_TODO" with status "Todo"
      And the integration has a task "E2E_TASK_IN_PROGRESS" with status "In Progress"
      Then the "Todo" column should contain the task card "E2E_TASK_TODO"
      And the "In Progress" column should contain the task card "E2E_TASK_IN_PROGRESS"

    Scenario: Each task card shows title, task type badge, and assignee avatar
      Given the integration has a task "E2E_CARD_TASK" with type "Story" assigned to "E2E_USER"
      Then the card for "E2E_CARD_TASK" should display the task title "E2E_CARD_TASK"
      And the card should show the task type badge "Story"
      And the card should show the assignee avatar for "E2E_USER"

    Scenario: Task cards with no assignee show an empty avatar placeholder
      Given the integration has an unassigned task "E2E_UNASSIGNED_TASK"
      Then the card for "E2E_UNASSIGNED_TASK" should show an unassigned icon

    Scenario: Columns with no tasks show an empty-state message
      Given the "Done" column contains no tasks
      Then the "Done" column should display an empty-state prompt

    Scenario: Clicking a task card opens the task detail panel or page
      Given the integration has a task "E2E_DETAIL_TASK"
      When the user clicks the card for "E2E_DETAIL_TASK"
      Then the task detail panel or page for "E2E_DETAIL_TASK" should open

  @authenticated
  Rule: Dragging tasks between board columns changes their status

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_DRAG_PROJECT" exists
      And the project has a "Product Backlog" integration with a "Board" view
      And the user has the "Edit Tasks" project permission in "E2E_DRAG_PROJECT"
      And the user has navigated to the "Product Backlog" board view inside "E2E_DRAG_PROJECT"

    Scenario: Dragging a task card to another column updates the task status
      Given the integration has a task "E2E_DRAG_TASK" in column "Todo"
      When the user drags "E2E_DRAG_TASK" to the "In Progress" column
      Then the task "E2E_DRAG_TASK" should appear in the "In Progress" column
      And the task's status should be updated to "In Progress"

    Scenario: Dropping a task back to its original column reverts the status
      Given the integration has a task "E2E_REVERT_TASK" in column "In Progress"
      When the user drags "E2E_REVERT_TASK" to the "In Progress" column and drops it there
      Then the task "E2E_REVERT_TASK" should remain in the "In Progress" column

    Scenario: User without "Edit Tasks" permission cannot drag task cards
      Given the user does not have the "Edit Tasks" project permission
      And the integration has a task "E2E_READONLY_TASK" in column "Todo"
      When the user attempts to drag "E2E_READONLY_TASK" to "In Progress"
      Then the drag operation should not be permitted
      And "E2E_READONLY_TASK" should remain in the "Todo" column

  @authenticated
  Rule: List view layout and task display

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_LIST_PROJECT" exists
      And the project has a "Product Backlog" integration with a "List" view
      And the user has the "View Sprints" project permission in "E2E_LIST_PROJECT"
      And the user has navigated to the "Product Backlog" list view inside "E2E_LIST_PROJECT"

    Scenario: List view renders tasks as rows grouped by status
      Then tasks should be displayed as rows grouped under their status heading
      And each group heading should show the status name and task count
      And each row should display the task title

    Scenario: Each list row shows columns for title, assignee, task type, priority, and status
      Given the integration has at least one task
      Then each task row should have visible columns for "Title", "Assignee", "Type", "Priority", and "Status"

    Scenario: Status groups in the list can be collapsed and expanded
      Given a status group "In Progress" is expanded and has tasks
      When the user clicks the "In Progress" group header toggle
      Then the "In Progress" group should collapse and hide its task rows
      When the user clicks the "In Progress" group header toggle again
      Then the "In Progress" group should expand and show its task rows

    Scenario: Clicking a task row opens the task detail panel or page
      Given the integration has a task "E2E_LIST_DETAIL_TASK"
      When the user clicks the row for "E2E_LIST_DETAIL_TASK"
      Then the task detail panel or page for "E2E_LIST_DETAIL_TASK" should open

    Scenario: Completed task groups are collapsed by default
      Given the integration has tasks with status "Done"
      Then the "Done" group should be collapsed by default
      And the collapsed heading should show the task count

  @authenticated
  Rule: Creating a task from the board view

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_CREATE_BOARD_PROJECT" exists
      And the project has a "Product Backlog" integration with a "Board" view
      And the user has the "Create Tasks" project permission in "E2E_CREATE_BOARD_PROJECT"
      And the user has navigated to the "Product Backlog" board view inside "E2E_CREATE_BOARD_PROJECT"

    Scenario: Each board column has an "Add task" button at the bottom
      Then every status column should show an "Add task" button at its bottom

    Scenario: Clicking "Add task" in a column opens an inline creation control
      When the user clicks "Add task" in the "Todo" column
      Then an inline task creation input should appear at the bottom of the "Todo" column

    Scenario: Typing a title and pressing Enter creates the task in that column
      When the user clicks "Add task" in the "Todo" column
      And the user types "E2E_BOARD_NEW_TASK" in the inline task input
      And the user presses Enter
      Then a task card named "E2E_BOARD_NEW_TASK" should appear in the "Todo" column
      And the inline input should close

    Scenario: Pressing Escape cancels inline task creation without creating a task
      When the user clicks "Add task" in the "Todo" column
      And the user types "E2E_CANCELLED_TASK" in the inline task input
      And the user presses Escape
      Then no task named "E2E_CANCELLED_TASK" should appear in the "Todo" column
      And the inline input should close

    Scenario: Submitting an empty task title does not create a task
      When the user clicks "Add task" in the "Backlog" column
      And the user presses Enter without typing a title
      Then no new task should be added to the "Backlog" column
      And the inline input should remain open or show a validation hint

    Scenario: User without "Create Tasks" permission does not see "Add task" buttons
      Given the user does not have the "Create Tasks" project permission
      Then no "Add task" button should be visible on any column

  @authenticated
  Rule: Creating a task from the list view

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_CREATE_LIST_PROJECT" exists
      And the project has a "Product Backlog" integration with a "List" view
      And the user has the "Create Tasks" project permission in "E2E_CREATE_LIST_PROJECT"
      And the user has navigated to the "Product Backlog" list view inside "E2E_CREATE_LIST_PROJECT"

    Scenario: Each status group in the list view has an "Add task" button
      Then every status group should show an "Add task" button below the last row

    Scenario: Clicking "Add task" in a group opens an inline creation row
      When the user clicks "Add task" in the "Todo" group
      Then an inline task creation row should appear at the bottom of the "Todo" group

    Scenario: Typing a title and pressing Enter creates the task in that group
      When the user clicks "Add task" in the "Backlog" group
      And the user types "E2E_LIST_NEW_TASK" in the inline task row
      And the user presses Enter
      Then a row named "E2E_LIST_NEW_TASK" should appear in the "Backlog" group
      And the inline row should close

    Scenario: Pressing Escape cancels inline creation in the list view
      When the user clicks "Add task" in the "Todo" group
      And the user types "E2E_LIST_CANCELLED_TASK" in the inline task row
      And the user presses Escape
      Then no task named "E2E_LIST_CANCELLED_TASK" should appear in the "Todo" group

    Scenario: User without "Create Tasks" permission does not see group "Add task" buttons
      Given the user does not have the "Create Tasks" project permission
      Then no "Add task" button should be visible in any status group

  @authenticated
  Rule: Managing views (create, rename, delete)

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_VM_PROJECT" exists
      And the project has a "Product Backlog" integration
      And the user has the "Manage Views" project permission in "E2E_VM_PROJECT"
      And the user has navigated to the "Product Backlog" integration inside "E2E_VM_PROJECT"

    Scenario: A "New view" button is visible in the view tab bar for authorised users
      Then a "New view" button or "+" control should be visible in the view tab bar

    Scenario: Clicking "New view" opens a view creation popover with layout options
      When the user clicks the "New view" button
      Then a popover or dialog should open
      And it should offer "Board" as a layout option
      And it should offer "List" as a layout option
      And it should contain a "View name" field pre-filled with a default name

    Scenario: Creating a Board view adds a new tab in the tab bar
      When the user clicks the "New view" button
      And the user changes the view name to "E2E_BOARD_VIEW"
      And the user selects the "Board" layout
      And the user confirms view creation
      Then a tab labelled "E2E_BOARD_VIEW" should appear in the view tab bar
      And the kanban board layout should be active

    Scenario: Creating a List view adds a new tab in the tab bar
      When the user clicks the "New view" button
      And the user changes the view name to "E2E_LIST_VIEW"
      And the user selects the "List" layout
      And the user confirms view creation
      Then a tab labelled "E2E_LIST_VIEW" should appear in the view tab bar
      And the tabular list layout should be active

    Scenario: Creating a view without a name defaults to a generated name
      When the user clicks the "New view" button
      And the user clears the view name field
      And the user selects the "Board" layout
      And the user confirms view creation
      Then a new tab should appear in the view tab bar using the generated default name

    Scenario: Renaming a view updates its tab label
      Given the integration has an existing view tab "E2E_OLD_VIEW_NAME"
      When the user right-clicks or opens the options menu for "E2E_OLD_VIEW_NAME"
      And the user selects "Rename view"
      And the user types "E2E_RENAMED_VIEW" and confirms
      Then the tab should be relabelled "E2E_RENAMED_VIEW"

    Scenario: Deleting a view removes its tab and activates the adjacent tab
      Given the integration has views "E2E_VIEW_ALPHA" and "E2E_VIEW_BETA"
      And the user is on the "E2E_VIEW_BETA" tab
      When the user opens the options menu for "E2E_VIEW_BETA" and selects "Delete view"
      And the user confirms the deletion
      Then the "E2E_VIEW_BETA" tab should no longer be visible
      And the "E2E_VIEW_ALPHA" tab should become active

    Scenario: The last remaining view cannot be deleted
      Given the integration has only one view "E2E_ONLY_VIEW"
      When the user opens the options menu for "E2E_ONLY_VIEW"
      Then the "Delete view" option should be disabled or absent

    Scenario: User without "Manage Views" permission does not see the "New view" button
      Given the user does not have the "Manage Views" project permission
      Then the "New view" button should not be visible in the view tab bar

  @authenticated
  Rule: Switching and persisting the active view

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_PERSIST_PROJECT" exists
      And the project has a "Product Backlog" integration with views "Board" and "List"
      And the user has the "View Sprints" project permission in "E2E_PERSIST_PROJECT"

    Scenario: Clicking a view tab switches the active layout
      Given the user is on the "Board" view tab inside "Product Backlog"
      When the user clicks the "List" view tab
      Then the tabular list layout should be displayed
      And the "List" tab should be marked as active

    Scenario: The active view tab is visually distinguished from inactive tabs
      When the user is on the "Board" view tab
      Then the "Board" tab should have a distinct active indicator
      And the "List" tab should not have the active indicator

    Scenario: Refreshing the page preserves the last active view
      Given the user has switched to the "List" view tab
      When the user refreshes the page
      Then the "List" view tab should still be active

  @authenticated
  Rule: Filtering and searching tasks within a view

    Background:
      Given the user already has a stored authenticated session
      And a project named "E2E_FILTER_PROJECT" exists
      And the project has a "Product Backlog" integration with a "Board" view
      And the user has the "View Sprints" project permission in "E2E_FILTER_PROJECT"
      And the user has navigated to the "Product Backlog" board view inside "E2E_FILTER_PROJECT"

    Scenario: A search or filter bar is visible at the top of the integration view
      Then a filter or search control should be visible in the view toolbar

    Scenario: Searching by keyword filters visible tasks across all columns
      Given the integration has tasks "E2E_ALPHA_TASK" and "E2E_BETA_TASK"
      When the user types "ALPHA" in the search bar
      Then only cards matching "ALPHA" should be visible
      And the card for "E2E_BETA_TASK" should not be visible

    Scenario: Filtering by assignee shows only tasks assigned to that user
      Given the integration has a task "E2E_OWNED_TASK" assigned to "E2E_FILTER_USER"
      And the integration has a task "E2E_OTHER_TASK" assigned to "E2E_OTHER_USER"
      When the user opens the filter panel and selects assignee "E2E_FILTER_USER"
      Then the card for "E2E_OWNED_TASK" should be visible
      And the card for "E2E_OTHER_TASK" should not be visible

    Scenario: Clearing the filter restores all tasks
      Given the user has applied a keyword filter "E2E_ALPHA"
      When the user clears the filter
      Then all tasks should be visible again
