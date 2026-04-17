// Package bootstrap wires up all application dependencies and exposes a
// runnable *App.
package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paca/api/internal/config"
	globalroledom "github.com/paca/api/internal/domain/globalrole"
	userdom "github.com/paca/api/internal/domain/user"
	"github.com/paca/api/internal/platform/authz"
	"github.com/paca/api/internal/platform/cache"
	"github.com/paca/api/internal/platform/database"
	"github.com/paca/api/internal/platform/logger"
	"github.com/paca/api/internal/platform/messaging"
	"github.com/paca/api/internal/platform/storage"
	jwttoken "github.com/paca/api/internal/platform/token"
	pgRepo "github.com/paca/api/internal/repository/postgres"
	redisRepo "github.com/paca/api/internal/repository/redis"
	attachmentsvc "github.com/paca/api/internal/service/attachment"
	authsvc "github.com/paca/api/internal/service/auth"
	globalrolesvc "github.com/paca/api/internal/service/globalrole"
	projectsvc "github.com/paca/api/internal/service/project"
	sprintsvc "github.com/paca/api/internal/service/sprint"
	tasksvc "github.com/paca/api/internal/service/task"
	usersvc "github.com/paca/api/internal/service/user"
	"github.com/paca/api/internal/transport/http/handler"
	"github.com/paca/api/internal/transport/http/router"
	"github.com/paca/api/internal/worker"
	"github.com/paca/api/migrations"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// App holds the HTTP server and any resources that need graceful shutdown.
type App struct {
	server           *http.Server
	publisher        *messaging.Publisher
	activityConsumer *worker.ActivityConsumer
	log              *slog.Logger
}

// New builds all dependencies and returns a ready-to-run App.
func New(cfg *config.Config) (*App, error) {
	log := logger.New(cfg.Env)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// --- Platform -----------------------------------------------------------
	db, err := database.Open(cfg.Database.DSN, log)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	redisClient, err := cache.NewClient(cfg.Redis.URL, log)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	publisher := messaging.NewPublisher(redisClient, log)

	tokenManager := jwttoken.New(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	permissionStore := pgRepo.NewAuthzPermissionStore(db)
	authorizer := authz.NewAuthorizer(permissionStore)

	// --- Repositories -------------------------------------------------------
	userRepo := pgRepo.NewUserRepository(db)
	globalRoleRepo := pgRepo.NewGlobalRoleRepository(db)
	projectRepo := pgRepo.NewProjectRepository(db)
	taskRepo := pgRepo.NewTaskRepository(db)
	activityRepo := pgRepo.NewTaskActivityRepository(db)
	sprintRepo := pgRepo.NewSprintRepository(db)
	viewRepo := pgRepo.NewViewRepository(db)
	attachmentRepo := pgRepo.NewAttachmentRepository(db)
	refreshStore := redisRepo.NewRefreshTokenStore(redisClient)

	// --- Schema migration (non-production only) -----------------------------
	// In development the embedded SQL migrations are run on every startup so
	// that a fresh database is always in the correct state without requiring
	// a manual migration step.  All statements use CREATE TABLE IF NOT EXISTS
	// / INSERT … ON CONFLICT so they are idempotent and safe to re-run.
	if cfg.Env != "production" {
		if err := database.RunMigrationsFS(db, migrations.FS); err != nil {
			return nil, fmt.Errorf("bootstrap: auto-migrate: %w", err)
		}
		log.Info("schema migrations applied")
	}

	// --- Admin seeding -------------------------------------------------------
	// seedDefaultRoles must run first so the ADMIN global role exists before
	// seedAdmin tries to reference it by FK.
	if err := seedDefaultRoles(context.Background(), db, userRepo, globalRoleRepo, cfg.Admin.Username, log); err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}
	if err := seedAdmin(context.Background(), userRepo, globalRoleRepo, cfg.Admin, log); err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	// --- Services -----------------------------------------------------------
	authService := authsvc.New(userRepo, tokenManager, refreshStore, cfg.JWT.RefreshTTL, cfg.JWT.RefreshSessionTTL)
	userService := usersvc.New(userRepo, permissionStore, globalRoleRepo)
	globalRoleService := globalrolesvc.New(globalRoleRepo)
	projectService := projectsvc.New(projectRepo, taskRepo)
	taskService := tasksvc.New(taskRepo)
	sprintService := sprintsvc.New(sprintRepo, taskRepo)
	viewService := sprintsvc.NewViewService(viewRepo)
	activityService := tasksvc.NewActivityService(activityRepo, projectRepo, publisher)
	activityConsumer := worker.NewActivityConsumer(redisClient, activityRepo, projectRepo, log)

	// Object storage — defaults to MinIO; switches to AWS S3 when STORAGE_PROVIDER=s3.
	storageClient, err := storage.NewS3Client(context.Background(), storage.S3Config{
		Endpoint:        cfg.Storage.Endpoint,
		PublicURL:       cfg.Storage.PublicURL,
		Region:          cfg.Storage.Region,
		Bucket:          cfg.Storage.Bucket,
		AccessKeyID:     cfg.Storage.AccessKeyID,
		SecretAccessKey: cfg.Storage.SecretAccessKey,
		UseSSL:          cfg.Storage.UseSSL,
		ForcePathStyle:  cfg.Storage.Provider != "s3", // MinIO requires path-style
	})
	if err != nil {
		return nil, fmt.Errorf("bootstrap: storage client: %w", err)
	}
	if err := storageClient.EnsureBucket(context.Background(), cfg.Storage.Bucket); err != nil {
		return nil, fmt.Errorf("bootstrap: ensure storage bucket: %w", err)
	}

	attachmentService := attachmentsvc.New(attachmentRepo, storageClient, cfg.Storage.Bucket)

	// --- Handlers -----------------------------------------------------------
	cookieCfg := handler.CookieConfig{
		Secure:            cfg.Server.CookieSecure,
		AccessTTL:         cfg.JWT.AccessTTL,
		RefreshTTL:        cfg.JWT.RefreshTTL,
		RefreshSessionTTL: cfg.JWT.RefreshSessionTTL,
	}

	deps := router.Deps{
		TokenManager: tokenManager,
		Authorizer:   authorizer,
		Health:       handler.NewHealthHandler(),
		Auth:         handler.NewAuthHandler(authService, cookieCfg),
		User:         handler.NewUserHandler(userService, authService),
		GlobalRole:   handler.NewGlobalRoleHandler(globalRoleService),
		Project:      handler.NewProjectHandler(projectService, authorizer, handler.WithProjectDefaultViews(viewService, taskService)),
		Task:         handler.NewTaskHandler(taskService, viewService, activityService),
		Sprint:       handler.NewSprintHandler(sprintService, viewService, handler.WithSprintDefaultTaskTypes(taskService)),
		View:         handler.NewViewHandler(viewService),
		Attachment:   handler.NewAttachmentHandler(attachmentService),
		Log:          log,
	}

	engine := router.New(deps)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{server: srv, publisher: publisher, activityConsumer: activityConsumer, log: log}, nil
}

// projectRoleModel is the GORM model used by seedDefaultProjectRoleTemplates
// to upsert canonical project-role permission sets on startup.
type projectRoleModel struct {
	ID          string  `gorm:"primarykey;type:uuid"`
	ProjectID   *string `gorm:"type:uuid;column:project_id;index"`
	RoleName    string  `gorm:"column:role_name;not null"`
	Permissions []byte  `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (projectRoleModel) TableName() string { return "project_roles" }

// Run starts the activity consumer and the HTTP server.
// It returns when the server stops.
func (a *App) Run() error {
	a.log.Info("starting server", "addr", a.server.Addr)
	a.activityConsumer.Start(context.Background())
	return a.server.ListenAndServe()
}

// Shutdown gracefully stops the server with the given timeout.
func (a *App) Shutdown(ctx context.Context) error {
	a.log.Info("shutting down server")
	a.activityConsumer.Stop()
	if a.publisher != nil {
		a.publisher.Close()
	}
	return a.server.Shutdown(ctx)
}

// seedAdmin ensures the default admin account exists in the database.
// It must be called after seedDefaultRoles so the ADMIN global role exists.
// If the account already exists it is left unchanged.
func seedAdmin(ctx context.Context, repo userdom.Repository, globalRoleRepo *pgRepo.GlobalRoleRepository, cfg config.AdminConfig, log *slog.Logger) error {
	_, err := repo.FindByUsernameIncludingDeleted(ctx, cfg.Username)
	if err == nil {
		// Admin already exists — nothing to do.
		return nil
	}
	if !errors.Is(err, userdom.ErrNotFound) {
		return fmt.Errorf("seed admin: lookup: %w", err)
	}

	// Resolve the ADMIN global role FK.
	adminRole, err := globalRoleRepo.FindByName(ctx, "ADMIN")
	if err != nil {
		return fmt.Errorf("seed admin: find ADMIN role: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seed admin: hash password: %w", err)
	}

	now := time.Now()
	admin := &userdom.User{
		ID:           uuid.New(),
		Username:     cfg.Username,
		PasswordHash: string(hash),
		FullName:     "Admin",
		RoleID:       adminRole.ID,
		Role:         adminRole.Name,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(ctx, admin); err != nil {
		return fmt.Errorf("seed admin: create: %w", err)
	}

	// Immediately assign the SUPER_ADMIN global role via users.role_id so the
	// admin user has full permissions from the first request.
	superAdminRole, err := globalRoleRepo.FindByName(ctx, "SUPER_ADMIN")
	if err != nil {
		return fmt.Errorf("seed admin: find SUPER_ADMIN role: %w", err)
	}
	if err := globalRoleRepo.ReplaceUserRoles(ctx, admin.ID, []uuid.UUID{superAdminRole.ID}); err != nil {
		return fmt.Errorf("seed admin: assign SUPER_ADMIN: %w", err)
	}

	log.Info("admin account created", "username", cfg.Username)
	return nil
}

func seedDefaultRoles(
	ctx context.Context,
	db *gorm.DB,
	userRepo userdom.Repository,
	globalRoleRepo *pgRepo.GlobalRoleRepository,
	adminUsername string,
	log *slog.Logger,
) error {
	for _, def := range authz.DefaultGlobalRoles() {
		role, err := globalRoleRepo.FindByName(ctx, def.Name)
		if err != nil {
			if !errors.Is(err, globalroledom.ErrNotFound) {
				return fmt.Errorf("seed global roles: find %s: %w", def.Name, err)
			}
			now := time.Now()
			if err := globalRoleRepo.Create(ctx, &globalroledom.GlobalRole{
				ID:          uuid.New(),
				Name:        def.Name,
				Permissions: permissionMap(def.Permissions),
				CreatedAt:   now,
				UpdatedAt:   now,
			}); err != nil {
				return fmt.Errorf("seed global roles: create %s: %w", def.Name, err)
			}
			continue
		}

		role.Permissions = permissionMap(def.Permissions)
		role.UpdatedAt = time.Now()
		if err := globalRoleRepo.Update(ctx, role); err != nil {
			return fmt.Errorf("seed global roles: update %s: %w", def.Name, err)
		}
	}

	if err := seedDefaultProjectRoleTemplates(ctx, db); err != nil {
		return err
	}

	adminUser, err := userRepo.FindByUsername(ctx, adminUsername)
	if err != nil {
		if errors.Is(err, userdom.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("seed global roles: load admin user: %w", err)
	}

	superAdminRole, err := globalRoleRepo.FindByName(ctx, "SUPER_ADMIN")
	if err != nil {
		return fmt.Errorf("seed global roles: load SUPER_ADMIN role: %w", err)
	}

	// Under the single-role schema users.role_id holds exactly one role.
	// Check whether the admin already has SUPER_ADMIN; if not, assign it (replacing whatever role they have).
	existingRoles, err := globalRoleRepo.ListUserRoles(ctx, adminUser.ID)
	if err != nil {
		return fmt.Errorf("seed global roles: list admin user roles: %w", err)
	}
	hasSuperAdmin := false
	for _, role := range existingRoles {
		if role.ID == superAdminRole.ID {
			hasSuperAdmin = true
			break
		}
	}
	if !hasSuperAdmin {
		if err := globalRoleRepo.ReplaceUserRoles(ctx, adminUser.ID, []uuid.UUID{superAdminRole.ID}); err != nil {
			return fmt.Errorf("seed global roles: assign SUPER_ADMIN: %w", err)
		}
		log.Info("assigned SUPER_ADMIN role to admin user", "username", adminUsername)
	}

	return nil
}

func seedDefaultProjectRoleTemplates(ctx context.Context, db *gorm.DB) error {
	for _, def := range authz.DefaultProjectRoles() {
		permissionsRaw, err := json.Marshal(permissionMap(def.Permissions))
		if err != nil {
			return fmt.Errorf("seed project roles: marshal %s permissions: %w", def.Name, err)
		}

		var existing projectRoleModel
		find := db.WithContext(ctx).
			Where("project_id IS NULL AND role_name = ?", def.Name).
			First(&existing)

		if errors.Is(find.Error, gorm.ErrRecordNotFound) {
			now := time.Now()
			projectRoleID := uuid.NewString()
			if err := db.WithContext(ctx).Create(&projectRoleModel{
				ID:          projectRoleID,
				ProjectID:   nil,
				RoleName:    def.Name,
				Permissions: permissionsRaw,
				CreatedAt:   now,
				UpdatedAt:   now,
			}).Error; err != nil {
				return fmt.Errorf("seed project roles: create template %s: %w", def.Name, err)
			}
			continue
		}
		if find.Error != nil {
			return fmt.Errorf("seed project roles: find template %s: %w", def.Name, find.Error)
		}

		if err := db.WithContext(ctx).
			Model(&projectRoleModel{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{"permissions": permissionsRaw, "updated_at": time.Now()}).Error; err != nil {
			return fmt.Errorf("seed project roles: update template %s: %w", def.Name, err)
		}
	}

	return nil
}

func permissionMap(permissions []authz.Permission) map[string]any {
	out := make(map[string]any, len(permissions))
	for _, p := range permissions {
		out[string(p)] = true
	}
	return out
}
