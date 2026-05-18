// Package router wires global middleware and all route groups onto a
// *gin.Engine.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Paca-AI/api/internal/platform/authz"
	jwttoken "github.com/Paca-AI/api/internal/platform/token"
	"github.com/Paca-AI/api/internal/transport/http/handler"
	httpmw "github.com/Paca-AI/api/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Deps holds all handler and middleware dependencies.
type Deps struct {
	TokenManager         *jwttoken.Manager
	APIKeyAuth           httpmw.APIKeyAuthenticator
	Authorizer           *authz.Authorizer
	ProjectVisibilitySvc httpmw.ProjectVisibilityChecker
	Health               *handler.HealthHandler
	Auth                 *handler.AuthHandler
	User                 *handler.UserHandler
	GlobalRole           *handler.GlobalRoleHandler
	Project              *handler.ProjectHandler
	Task                 *handler.TaskHandler
	Sprint               *handler.SprintHandler
	View                 *handler.ViewHandler
	Attachment           *handler.AttachmentHandler
	Document             *handler.DocumentHandler
	DocFile              *handler.DocFileHandler
	Notification         *handler.NotificationHandler
	APIKey               *handler.APIKeyHandler
	Plugin               *handler.PluginHandler
	Log                  *slog.Logger
}

// New builds and returns a configured *gin.Engine.
func New(deps Deps) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(requestIDMiddleware())
	r.Use(loggerMiddleware(deps.Log))
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	api := r.Group("/api")

	// Public routes
	api.GET("/healthz", deps.Health.Check)

	v1 := api.Group("/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", deps.Auth.Login)
			auth.POST("/refresh", deps.Auth.Refresh)
			auth.POST("/logout", httpmw.Authn(deps.TokenManager), deps.Auth.Logout)
		}

		users := v1.Group("/users")
		users.Use(httpmw.Authn(deps.TokenManager, deps.APIKeyAuth))
		{
			// Password change is allowed even when MustChangePassword=true so
			// that users can fulfil the forced-change requirement.
			users.PATCH("/me/password", httpmw.RequireJWTAuth(), deps.User.ChangeMyPassword)

			// All other self-service routes require a fresh (non-forced) password.
			me := users.Group("")
			me.Use(httpmw.RequireFreshPassword())
			{
				me.GET("/me", deps.User.GetMe)
				me.PATCH("/me", deps.User.UpdateMe)
				me.GET("/me/global-permissions", deps.User.GetMyGlobalPermissions)

				// API key management -- requires JWT/cookie session auth; API key
				// credentials are explicitly rejected to prevent privilege escalation
				// via a leaked API key.
				if deps.APIKey != nil {
					apiKeys := me.Group("")
					apiKeys.Use(httpmw.RequireJWTAuth())
					apiKeys.GET("/me/api-keys", deps.APIKey.List)
					apiKeys.POST("/me/api-keys", deps.APIKey.Create)
					apiKeys.DELETE("/me/api-keys/:keyId", deps.APIKey.Revoke)
				}

				// Notification routes
				if deps.Notification != nil {
					me.GET("/me/notifications", deps.Notification.List)
					me.PATCH("/me/notifications/:notificationId/read", deps.Notification.MarkAsRead)
					me.POST("/me/notifications/read-all", deps.Notification.MarkAllAsRead)
				}
			}
		}

		admin := v1.Group("/admin")
		admin.Use(httpmw.Authn(deps.TokenManager, deps.APIKeyAuth))
		admin.Use(httpmw.RequireFreshPassword())
		{
			// User management — requires users.* permissions
			admin.GET("/users",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersRead),
				deps.User.ListUsers,
			)
			admin.POST("/users",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersWrite),
				deps.User.CreateUser,
			)
			admin.GET("/users/:userId",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersRead),
				deps.User.GetUserByID,
			)
			admin.PATCH("/users/:userId",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersWrite),
				deps.User.AdminUpdateUser,
			)
			admin.PATCH("/users/:userId/password",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersWrite),
				deps.User.ResetPassword,
			)
			admin.DELETE("/users/:userId",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersDelete),
				deps.User.DeleteUser,
			)

			// Global role management — requires global_roles.* permissions
			admin.GET("/global-roles",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionGlobalRolesRead),
				deps.GlobalRole.List,
			)
			admin.POST("/global-roles",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionGlobalRolesWrite),
				deps.GlobalRole.Create,
			)
			admin.PATCH("/global-roles/:roleId",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionGlobalRolesWrite),
				deps.GlobalRole.Update,
			)
			admin.DELETE("/global-roles/:roleId",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionGlobalRolesWrite),
				deps.GlobalRole.Delete,
			)
			admin.PUT("/users/:userId/global-roles",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionGlobalRolesAssign),
				deps.GlobalRole.ReplaceUserRoles,
			)
		}

		// Project routes — list/create are collection-level; all other actions are
		// project-scoped and accessible to members with appropriate roles.
		// Collection routes require authentication; per-project read routes also
		// allow anonymous access when the project has is_public = true.
		projects := v1.Group("/projects")
		projects.Use(httpmw.Authn(deps.TokenManager, deps.APIKeyAuth))
		projects.Use(httpmw.RequireFreshPassword())
		{
			// Collection routes
			// ListProjects self-selects: global projects.read → all projects,
			// otherwise → only the caller's accessible projects.
			projects.GET("", deps.Project.ListProjects)
			projects.POST("",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionProjectsCreate),
				deps.Project.CreateProject,
			)
		}

		// Single-project routes — use optional auth so anonymous users can access
		// public projects (is_public = true) without credentials.
		project := v1.Group("/projects/:projectId")
		project.Use(httpmw.OptionalAuthn(deps.TokenManager, deps.APIKeyAuth))
		project.Use(httpmw.RequireFreshPassword())
		{
			project.GET("",
				// Allow: global projects.read OR project-scoped projects.read OR public project.
				httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
					httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
					httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
				),
				deps.Project.GetProject,
			)
			project.PATCH("",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectsWrite),
				deps.Project.UpdateProject,
			)
			project.DELETE("",
				httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectsDelete),
				deps.Project.DeleteProject,
			)

			members := project.Group("/members")
			{
				members.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionProjectMembersRead}},
					),
					deps.Project.ListMembers,
				)
				members.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectMembersWrite),
					deps.Project.AddMember,
				)
				// Static sub-path must come before /:userId to avoid the param swallowing it.
				members.GET("/me/permissions", deps.Project.GetMyProjectPermissions)
				members.PATCH("/:userId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectMembersWrite),
					deps.Project.UpdateMemberRole,
				)
				members.DELETE("/:userId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectMembersWrite),
					deps.Project.RemoveMember,
				)
			}

			roles := project.Group("/roles")
			{
				roles.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionProjectRolesRead}},
					),
					deps.Project.ListRoles,
				)
				roles.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectRolesWrite),
					deps.Project.CreateRole,
				)
				roles.PATCH("/:roleId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectRolesWrite),
					deps.Project.UpdateRole,
				)
				roles.DELETE("/:roleId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionProjectRolesWrite),
					deps.Project.DeleteRole,
				)
			}

			// Task types — project-scoped configuration
			taskTypes := project.Group("/task-types")
			{
				taskTypes.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.ListTaskTypes,
				)
				taskTypes.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.CreateTaskType,
				)
				taskTypes.PATCH("/:typeId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.UpdateTaskType,
				)
				taskTypes.DELETE("/:typeId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.DeleteTaskType,
				)
				taskTypes.PUT("/:typeId/set-default",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.SetDefaultTaskType,
				)
			}

			// Task statuses — project-scoped workflow configuration
			taskStatuses := project.Group("/task-statuses")
			{
				taskStatuses.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.ListTaskStatuses,
				)
				taskStatuses.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.CreateTaskStatus,
				)
				taskStatuses.PATCH("/:statusId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.UpdateTaskStatus,
				)
				taskStatuses.DELETE("/:statusId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.DeleteTaskStatus,
				)
				taskStatuses.PUT("/:statusId/set-default",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.SetDefaultTaskStatus,
				)
			}

			// Sprints
			sprints := project.Group("/sprints")
			{
				sprints.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionSprintsRead}},
					),
					deps.Sprint.ListSprints,
				)
				sprints.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.Sprint.CreateSprint,
				)
				sprints.GET("/:sprintId",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionSprintsRead}},
					),
					deps.Sprint.GetSprint,
				)
				sprints.PATCH("/:sprintId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.Sprint.UpdateSprint,
				)
				sprints.DELETE("/:sprintId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.Sprint.DeleteSprint,
				)
				sprints.POST("/:sprintId/complete",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.Sprint.CompleteSprint,
				)
			}

			// Views — unified endpoint for sprint, backlog, and timeline views.
			// Use ?context=sprint|backlog|timeline; sprint context also requires ?sprint_id=<uuid>.
			views := project.Group("/views")
			{
				views.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionSprintsRead}},
					),
					deps.View.ListViews,
				)
				views.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.View.CreateView,
				)
				// Static /positions must be registered before /:viewId.
				views.PUT("/positions",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.View.ReorderViews,
				)
				views.GET("/:viewId",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionSprintsRead}},
					),
					deps.View.GetView,
				)
				views.PATCH("/:viewId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.View.UpdateView,
				)
				views.DELETE("/:viewId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionSprintsWrite),
					deps.View.DeleteView,
				)
				views.GET("/:viewId/task-positions",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.View.ListTaskPositions,
				)
				views.PUT("/:viewId/task-positions",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.View.BulkMoveTasks,
				)
				views.PUT("/:viewId/task-positions/:taskId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.View.MoveTask,
				)
			}

			// Tasks — core work items
			tasks := project.Group("/tasks")
			{
				tasks.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.ListTasks,
				)
				tasks.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.CreateTask,
				)
				tasks.GET("/by-number/:taskNumber",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.GetTaskByNumber,
				)
				tasks.GET("/:taskId",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.GetTask,
				)
				tasks.PATCH("/:taskId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.UpdateTask,
				)
				tasks.DELETE("/:taskId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.DeleteTask,
				)

				// Activities — task activity log and user comments
				activities := tasks.Group("/:taskId/activities")
				{
					activities.GET("",
						httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
							httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
							httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
						),
						deps.Task.ListTaskActivities,
					)
					// Comments are a sub-resource of activities
					comments := activities.Group("/comments")
					{
						comments.POST("",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
							deps.Task.AddComment,
						)
						comments.PATCH("/:commentId",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
							deps.Task.UpdateComment,
						)
						comments.DELETE("/:commentId",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
							deps.Task.DeleteComment,
						)
					}
				}

				// Attachments — files uploaded and linked to a task
				attachments := tasks.Group("/:taskId/attachments")
				{
					attachments.GET("",
						httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
							httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
							httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
						),
						deps.Attachment.ListTaskAttachments,
					)
					attachments.POST("/initiate-upload",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
						deps.Attachment.InitiateUpload,
					)
					attachments.POST("/complete-upload",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
						deps.Attachment.CompleteUpload,
					)
					attachments.GET("/:attachmentId/download-url",
						httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
							httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
							httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
						),
						deps.Attachment.GetDownloadURL,
					)
					attachments.DELETE("/:attachmentId",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
						deps.Attachment.DeleteTaskAttachment,
					)
				}
			}

			// Custom field definitions — project-level schema
			customFields := project.Group("/custom-fields")
			{
				customFields.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.ListCustomFieldDefinitions,
				)
				customFields.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.CreateCustomFieldDefinition,
				)
				customFields.GET("/:fieldId",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionTasksRead}},
					),
					deps.Task.GetCustomFieldDefinition,
				)
				customFields.PATCH("/:fieldId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.UpdateCustomFieldDefinition,
				)
				customFields.DELETE("/:fieldId",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionTasksWrite),
					deps.Task.DeleteCustomFieldDefinition,
				)
			}

			// Documentation — project documents with folder hierarchy, snapshots, and activity
			docs := project.Group("/docs")
			{
				// Folders
				folders := docs.Group("/folders")
				{
					folders.GET("",
						httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
							httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
							httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
						),
						deps.Document.ListFolders,
					)
					folders.POST("",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
						deps.Document.CreateFolder,
					)
					folders.PATCH("/:folderId",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
						deps.Document.UpdateFolder,
					)
					folders.DELETE("/:folderId",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
						deps.Document.DeleteFolder,
					)
				}

				// Documents — collection
				docs.GET("",
					httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
						httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
						httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
					),
					deps.Document.ListDocuments,
				)
				docs.POST("",
					httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
					deps.Document.CreateDocument,
				)

				// Documents — single item
				doc := docs.Group("/:docId")
				{
					doc.GET("",
						httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
							httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
							httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
						),
						deps.Document.GetDocument,
					)
					doc.PATCH("",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
						deps.Document.UpdateDocument,
					)
					doc.DELETE("",
						httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
						deps.Document.DeleteDocument,
					)

					// Snapshots — version history
					snapshots := doc.Group("/snapshots")
					{
						snapshots.GET("",
							httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
								httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
								httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
							),
							deps.Document.ListSnapshots,
						)
						snapshots.GET("/:snapshotId",
							httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
								httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
								httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
							),
							deps.Document.GetSnapshot,
						)
					}

					// Activity log and comments
					doc.GET("/activities",
						httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
							httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
							httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
						),
						deps.Document.ListActivities,
					)
					comments := doc.Group("/comments")
					{
						comments.POST("",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
							deps.Document.AddComment,
						)
						comments.PATCH("/:commentId",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
							deps.Document.UpdateComment,
						)
						comments.DELETE("/:commentId",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
							deps.Document.DeleteComment,
						)
					}

					// Doc file uploads — stored directly in the files table (no join table)
					docFiles := doc.Group("/files")
					{
						docFiles.POST("/initiate-upload",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
							deps.DocFile.InitiateDocUpload,
						)
						docFiles.POST("/complete-upload",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
							deps.DocFile.CompleteDocUpload,
						)
						docFiles.GET("/:fileId/download-url",
							httpmw.RequirePublicProjectOrPermissions(deps.ProjectVisibilitySvc, deps.Authorizer,
								httpmw.PermissionGroup{Scope: httpmw.GlobalScope(), Permissions: []authz.Permission{authz.PermissionProjectsRead}},
								httpmw.PermissionGroup{Scope: httpmw.ProjectScopeFromParam("projectId"), Permissions: []authz.Permission{authz.PermissionDocsRead}},
							),
							deps.DocFile.GetDocFileDownloadURL,
						)
						docFiles.DELETE("/:fileId",
							httpmw.RequirePermissions(deps.Authorizer, httpmw.ProjectScopeFromParam("projectId"), authz.PermissionDocsWrite),
							deps.DocFile.DeleteDocFile,
						)
					}
				}
			}

		}

		// Plugin routes — management (admin), extension settings (admin), and proxy (per-plugin).
		if deps.Plugin != nil {
			// Public listing: any authenticated user can see installed plugins, anonymous users can also access.
			pluginList := v1.Group("/plugins")
			pluginList.Use(httpmw.OptionalAuthn(deps.TokenManager, deps.APIKeyAuth))
			pluginList.Use(httpmw.RequireFreshPassword())
			pluginList.GET("", deps.Plugin.ListPlugins)

			// Plugin proxy routes — forward requests to plugin WASM handlers.
			// The full sub-path (including any /projects/:projectId/ segment) is
			// captured by the wildcard and matched against the plugin's own route
			// manifest. Route-level authn/authz is enforced by the plugin proxy
			// handler based on the per-route middleware declarations in the manifest.
			v1.Any("/plugins/:pluginId/*path", deps.Plugin.ProxyRequest)

			// Admin plugin management — requires global users.write permission
			// (no dedicated plugin permission exists yet).
			adminPlugins := v1.Group("/admin/plugins")
			adminPlugins.Use(httpmw.Authn(deps.TokenManager, deps.APIKeyAuth))
			adminPlugins.Use(httpmw.RequireFreshPassword())
			adminPlugins.Use(httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersWrite))
			{
				adminPlugins.GET("/marketplace", deps.Plugin.ListMarketplacePlugins)
				adminPlugins.POST("/marketplace/install", deps.Plugin.InstallMarketplacePlugin)
				adminPlugins.POST("", deps.Plugin.InstallPlugin)
				adminPlugins.PATCH("/:pluginId", deps.Plugin.UpdatePlugin)
				adminPlugins.POST("/:pluginId/upgrade", deps.Plugin.UpgradeMarketplacePlugin)
				adminPlugins.DELETE("/:pluginId", deps.Plugin.DeletePlugin)
			}

			// Admin extension settings — system-wide ordering/visibility, super admin only.
			adminExtSettings := v1.Group("/admin/plugin-extension-settings")
			adminExtSettings.Use(httpmw.Authn(deps.TokenManager, deps.APIKeyAuth))
			adminExtSettings.Use(httpmw.RequireFreshPassword())
			adminExtSettings.Use(httpmw.RequirePermissions(deps.Authorizer, httpmw.GlobalScope(), authz.PermissionUsersWrite))
			{
				adminExtSettings.PATCH("", deps.Plugin.UpdateExtensionSetting)
			}
		}
	}

	return r
}

// requestIDMiddleware attaches a UUID request ID to every request context and
// response header.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// loggerMiddleware logs method, path, status, and latency via slog.
func loggerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetString("request_id"),
		)
	}
}

// corsMiddleware sets permissive CORS headers (tighten in production).
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
