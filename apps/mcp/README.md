# Paca MCP Server

Model Context Protocol (MCP) server for Paca - an open-source, AI-native project management platform.

## Overview

This MCP server provides access to Paca's API through the Model Context Protocol, allowing AI assistants to interact with Paca projects, tasks, sprints, and documents using API key authentication.

## Features

- **API Key Authentication**: Secure access using Paca API keys
- **Comprehensive Project Management**: Full project lifecycle with member and role management
- **Advanced Task Management**: Tasks with types, statuses, custom fields, attachments, and BDD scenarios
- **Sprint Management**: Complete sprint lifecycle management
- **Document Management**: Documents with folder hierarchy, version history, and file support
- **View Management**: Multiple view types (sprint, backlog, timeline) with task positioning
- **GitHub Integration**: Repository linking, PR management, and branch creation
- **Collaboration Tools**: Comments, activities, and BDD scenarios for team collaboration
- **BlockNote Integration**: Automatic conversion between BlockNote JSON and Markdown
- **Modular Architecture**: Clean separation of concerns for easy maintenance and contribution

## Installation

1. Clone the repository and navigate to the MCP server directory:
   ```bash
   cd apps/mcp
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Build the server:
   ```bash
   npm run build
   ```

## Configuration

Set the following environment variables:

```bash
export PACA_API_KEY="your-api-key-here"
export PACA_API_URL="http://localhost:8080"  # Default: http://localhost:8080
```

## Available Tools

The MCP server provides **86 tools** across **17 categories** for comprehensive project management.

### 📁 Project Management (5 tools)
- `list_projects` - List all accessible projects
- `get_project` - Get details of a specific project
- `create_project` - Create a new project
- `update_project` - Update an existing project
- `delete_project` - Delete a project

### ✅ Task Management (6 tools)
- `list_tasks` - List all tasks in a project
- `get_task` - Get details of a specific task
- `get_task_by_number` - Get a task by its number
- `create_task` - Create a new task
- `update_task` - Update an existing task
- `delete_task` - Delete a task

### 🏃 Sprint Management (6 tools)
- `list_sprints` - List all sprints in a project
- `get_sprint` - Get details of a specific sprint
- `create_sprint` - Create a new sprint
- `update_sprint` - Update an existing sprint
- `delete_sprint` - Delete a sprint
- `complete_sprint` - Mark a sprint as completed

### 📄 Document Management (5 tools)
- `list_documents` - List all documents in a project
- `get_document` - Get details of a specific document
- `create_document` - Create a new document
- `update_document` - Update an existing document
- `delete_document` - Delete a document

### 👥 Project Members (5 tools)
- `list_project_members` - List all members of a project
- `add_project_member` - Add a member to a project
- `get_my_project_permissions` - Get the current user's permissions
- `update_project_member_role` - Update a member's role
- `remove_project_member` - Remove a member from a project

### 🎭 Project Roles (4 tools)
- `list_project_roles` - List all roles in a project
- `create_project_role` - Create a new project role
- `update_project_role` - Update an existing project role
- `delete_project_role` - Delete a project role

### 🏷️ Task Types (5 tools)
- `list_task_types` - List all task types in a project
- `create_task_type` - Create a new task type
- `update_task_type` - Update an existing task type
- `delete_task_type` - Delete a task type
- `set_default_task_type` - Set a task type as default

### 📊 Task Statuses (4 tools)
- `list_task_statuses` - List all task statuses in a project
- `create_task_status` - Create a new task status
- `update_task_status` - Update an existing task status
- `delete_task_status` - Delete a task status

### 🎯 Views (9 tools)
- `list_views` - List all views in a project
- `create_view` - Create a new view (sprint/backlog/timeline)
- `reorder_views` - Reorder views in a project
- `get_view` - Get details of a specific view
- `update_view` - Update an existing view
- `delete_view` - Delete a view
- `list_task_positions` - List task positions in a view
- `bulk_move_tasks` - Bulk move tasks in a view
- `move_task` - Move a task within a view

### 🔧 Custom Fields (5 tools)
- `list_custom_fields` - List all custom field definitions
- `create_custom_field` - Create a new custom field definition
- `get_custom_field` - Get details of a custom field
- `update_custom_field` - Update a custom field definition
- `delete_custom_field` - Delete a custom field definition

### 📎 Attachments (3 tools)
- `list_task_attachments` - List all attachments for a task
- `get_attachment_download_url` - Get a download URL for an attachment
- `delete_task_attachment` - Delete an attachment

### 🧪 BDD Scenarios (5 tools)
- `list_bdd_scenarios` - List all BDD scenarios for a task
- `create_bdd_scenario` - Create a new BDD scenario (Given/When/Then)
- `get_bdd_scenario` - Get details of a BDD scenario
- `update_bdd_scenario` - Update an existing BDD scenario
- `delete_bdd_scenario` - Delete a BDD scenario

### 📁 Document Folders (4 tools)
- `list_doc_folders` - List all folders in a project
- `create_doc_folder` - Create a new document folder
- `update_doc_folder` - Update a document folder
- `delete_doc_folder` - Delete a document folder

### 📸 Document Snapshots (2 tools)
- `list_doc_snapshots` - List all snapshots of a document
- `get_doc_snapshot` - Get a specific document snapshot

### 🔗 GitHub Integration (7 tools)
- `get_github_integration` - Get GitHub integration status
- `set_github_token` - Set GitHub token for a project
- `delete_github_token` - Delete GitHub token
- `list_github_repositories` - List available GitHub repositories
- `list_linked_github_repos` - List linked repositories
- `link_github_repository` - Link a GitHub repository
- `unlink_github_repository` - Unlink a GitHub repository

### 💬 Task Activities (4 tools)
- `list_task_activities` - List all activities for a task
- `add_task_comment` - Add a comment to a task
- `update_task_comment` - Update a task comment
- `delete_task_comment` - Delete a task comment

### 🔀 Task GitHub (5 tools)
- `list_task_prs` - List pull requests linked to a task
- `link_pr_to_task` - Link a pull request to a task
- `unlink_pr_from_task` - Unlink a pull request
- `create_branch_for_task` - Create a branch for a task
- `list_task_branches` - List branches for a task

For a complete list of all tools with detailed descriptions, see [ALL_TOOLS.md](./ALL_TOOLS.md).

## Architecture

The project follows a modular architecture with clear separation of concerns:

```
src/
├── api/          # API client and HTTP communication
├── tools/        # MCP tool definitions and handlers
├── types/        # TypeScript type definitions
├── utils/        # Utility functions (converters, formatters)
├── server.ts     # MCP server setup
└── index.ts      # Entry point
```

For detailed architecture information, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## Markdown/BlockNote Conversion

The MCP server automatically handles conversion between Markdown and BlockNote JSON format:

- **Reading**: When fetching tasks or documents with descriptions/content, the server converts BlockNote JSON to Markdown for easier readability
- **Writing**: When creating or updating tasks or documents with descriptions/content, the server converts Markdown to BlockNote JSON format

This allows AI assistants to work with familiar Markdown format while the API stores content in BlockNote's rich text format.

## Usage with MCP Clients

### Claude Desktop

Add the following to your Claude Desktop config:

```json
{
  "mcpServers": {
    "paca": {
      "command": "node",
      "args": ["/path/to/paca/apps/mcp/build/index.js"],
      "env": {
        "PACA_API_KEY": "your-api-key-here",
        "PACA_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Other MCP Clients

The server can be run with any MCP-compatible client by providing:
1. The path to the built server executable
2. Required environment variables (PACA_API_KEY, optionally PACA_API_URL)

## Development

### Build

```bash
npm run build
```

### Watch Mode

For development with auto-rebuild:
```bash
npm run watch
```

### MCP Inspector

To test the server with the MCP Inspector:
```bash
npm run inspector
```

### Project Structure

For detailed information about the codebase structure and how to contribute, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## API Key Support

All endpoints in this MCP server support API key authentication. The server uses the `X-API-Key` header for authentication, which is compatible with Paca's API key system.

## Examples

### Create a Task with Markdown Description

```
Tool: create_task
Arguments:
- projectId: "project-uuid"
- title: "Implement user authentication"
- description: "# Implementation Plan\n\n## Steps\n1. Create auth service\n2. Add login endpoint\n3. Implement JWT tokens"
- statusId: "status-uuid"
- importance: 5
- tags: ["auth", "backend"]
```

### Update Document Content

```
Tool: update_document
Arguments:
- projectId: "project-uuid"
- docId: "doc-uuid"
- content: "# System Design\n\n## Architecture\nThis document describes the..."
```

## Notes

- The server requires a running Paca API instance
- API keys can be created through the Paca web interface under user settings
- All descriptions and document contents are automatically converted between Markdown and BlockNote format
- Date fields should be provided in ISO 8601 format (e.g., "2024-01-01T00:00:00Z")

## Contributing

We welcome contributions! Please see [DEVELOPMENT.md](./DEVELOPMENT.md) for information about:
- Code architecture and structure
- How to add new tools
- Code style guidelines
- Testing procedures

## License

Apache License 2.0
