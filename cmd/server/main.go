package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/markbates/goth"
	"github.com/thetkpark/cscms-temp-storage/data"
	"github.com/thetkpark/cscms-temp-storage/handlers"
	"github.com/thetkpark/cscms-temp-storage/router"
	"github.com/thetkpark/cscms-temp-storage/service/encrypt"
	"github.com/thetkpark/cscms-temp-storage/service/jwt"
	"github.com/thetkpark/cscms-temp-storage/service/storage"
	"github.com/thetkpark/cscms-temp-storage/service/token"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/arsmn/fiber-swagger/v2"
	"github.com/caarlos0/env/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/shareed2k/goth_fiber"
	_ "github.com/thetkpark/cscms-temp-storage/cmd/server/docs"
)

// @title CSCMS Storage
// @version 1.0
// @description This is documentation for CSCMS Storage API
func main() {
	appENVs := ApplicationEnvironmentVariable{}
	if err := env.Parse(&appENVs, env.Options{RequiredIfNoDef: true}); err != nil {
		log.Fatalln("Failed to get app ENVs: ", err)
	}

	zapLogger, _ := zap.NewProduction()
	if appENVs.Env == "development" {
		zapLogger, _ = zap.NewDevelopment()
	}
	logger := zapLogger.Sugar()
	defer func(zapLogger *zap.Logger) {
		err := zapLogger.Sync()
		if err != nil {
			logger.Errorw("unable to flush logger", "error", err)
		}
	}(zapLogger)

	// Create file in /tmp for livenessProbe
	livenessProbeFilePath := "/tmp/cscms-storage-healthy"
	if _, err := os.Create(livenessProbeFilePath); err != nil {
		logger.Fatalw("Unable to create livenessProbe file", "error", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			logger.Errorw("unable to delete liveness probe file", "error", err)
		}
	}(livenessProbeFilePath)

	app := router.NewFiberRouter()

	// Create data store
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", appENVs.DB.Username, appENVs.DB.Password, appENVs.DB.Host, appENVs.DB.Port, appENVs.DB.DatabaseName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatalw("unable to open sqlite db", "error", err)
	}
	gormFileDataStore, err := data.NewGormFileDataStore(db, time.Duration(appENVs.FileStoreMaxDuration)*time.Hour*24)
	if err != nil {
		logger.Fatalw("unable to run gorm migration on file table", "error", err)
	}
	gormImageDataStore, err := data.NewGormImageDataStore(db)
	if err != nil {
		logger.Fatalw("unable to run gorm migration on image table", "error", err)
	}
	gormUserDataStore, err := data.NewGormUserDataStore(db)
	if err != nil {
		logger.Fatalw("unable to run gorm migration on user table", "error", err)
	}

	// Create service managers for handler
	sioEncryptionManager := encrypt.NewSIOEncryptionManager(logger, appENVs.MasterKey)
	diskStorageManager, err := storage.NewDiskStorageManager(logger, appENVs.FileStoragePath)
	if err != nil {
		logger.Fatalw("unable to create disk storage manager", "error", err)
	}
	imageStorageManager, err := storage.NewAzureImageStorageManager(logger, appENVs.AzureBlobStorageConnectionString, appENVs.AzureBlobStorageContainerName)
	if err != nil {
		logger.Fatalw("unable to azure image storage manager", "error", err)
	}
	jwtManager := jwt.NewJWTManager(appENVs.JWTSecret)
	tokenManager := token.NewNanoIDTokenManager()

	// Create handlers
	fileHandler := handlers.NewFileRoutesHandler(logger, sioEncryptionManager, gormFileDataStore, diskStorageManager, tokenManager, time.Duration(appENVs.FileStoreMaxDuration)*time.Hour*24)
	imageHandler := handlers.NewImageRouteHandler(logger, gormImageDataStore, imageStorageManager, tokenManager)
	authHandler := handlers.NewAuthRouteHandler(logger, gormUserDataStore, jwtManager, tokenManager, appENVs.Entrypoint)

	app.Use(limiter.New(limiter.Config{
		Expiration: time.Second * 5,
		Max:        10,
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://storage.cscms.me, http://localhost:3000",
		AllowMethods:     "GET POST PATCH DELETE",
		AllowCredentials: true,
	}))
	app.Use(compress.New(compress.Config{
		Next: func(c *fiber.Ctx) bool {
			t := c.Params("token", "")
			return len(t) == 0
		},
		Level: compress.LevelBestSpeed,
	}))
	//app.Use(csrf.New(csrf.Config{
	//}))

	app.Get("/api/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":   "pong",
			"timestamp": time.Now(),
		})
	})

	apiPath := app.Group("/api", authHandler.ParseUser)

	filePath := apiPath.Group("/file")
	filePath.Post("/", fileHandler.UploadFile)
	filePath.Get("/", authHandler.AuthenticatedOnly, fileHandler.GetOwnFiles)
	filePath.Patch("/:fileID", authHandler.AuthenticatedOnly, fileHandler.IsOwnFile, fileHandler.EditToken)
	filePath.Delete("/:fileID", authHandler.AuthenticatedOnly, fileHandler.IsOwnFile, fileHandler.DeleteFile)

	imagePath := apiPath.Group("/image")
	imagePath.Post("/", imageHandler.UploadImage)
	imagePath.Get("/", authHandler.AuthenticatedOnly, imageHandler.GetOwnImages)
	imagePath.Delete("/:imageID", authHandler.AuthenticatedOnly, imageHandler.IsOwnImage, imageHandler.DeleteImage)

	// User Authentication with Oauth
	goth.UseProviders(
		github.New(appENVs.OauthGitHubClientSecret, appENVs.OauthGitHubSecretKey, fmt.Sprintf("%s/auth/github/callback", appENVs.Entrypoint), "user:email"),
		google.New(appENVs.OAuthGoogleClientSecret, appENVs.OAuthGoogleSecretKey, fmt.Sprintf("%s/auth/google/callback", appENVs.Entrypoint), "email", "profile"))

	authPath := app.Group("/auth")
	authPath.Get("/logout", authHandler.Logout)
	authPath.Get("/user", authHandler.ParseUser, authHandler.AuthenticatedOnly, authHandler.GetUserInfo)
	authPath.Get("/:provider", goth_fiber.BeginAuthHandler)
	authPath.Get("/:provider/callback", authHandler.OauthProviderCallback)
	apiPath.Post("/auth/token", authHandler.ParseUser, authHandler.AuthenticatedOnly, authHandler.GenerateAPIToken)

	// Other routes
	app.Static("/", "./client/build")
	app.Static("/404", "./client/build")
	app.Get("/swagger/*", swagger.Handler)
	app.Get("/:token", fileHandler.GetFile)

	// Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		_ = <-sigChan
		logger.Info("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	if err := app.Listen(fmt.Sprintf(":%s", appENVs.Port)); err != nil {
		logger.Fatalw(fmt.Sprintf("unable to start server on %s", appENVs.Port), "error", err)
	}
}

type ApplicationEnvironmentVariable struct {
	MasterKey                        string `env:"MASTER_KEY"`
	FileStoragePath                  string `env:"STORAGE_PATH"`
	FileStoreMaxDuration             int    `env:"STORE_DURATION" envDefault:"30"`
	AzureBlobStorageConnectionString string `env:"AZSTORAGE_CONNECTION_STRING"`
	AzureBlobStorageContainerName    string `env:"AZSTORAGE_CONTAINER_NAME"`
	Port                             string `env:"PORT"`
	DB                               DatabaseEnvironmentVariable
	OauthGitHubClientSecret          string `env:"GITHUB_OAUTH_CLIENT_ID"`
	OauthGitHubSecretKey             string `env:"GITHUB_OAUTH_SECRET_KEY"`
	OAuthGoogleClientSecret          string `env:"GOOGLE_OAUTH_CLIENT_ID"`
	OAuthGoogleSecretKey             string `env:"GOOGLE_OAUTH_SECRET_KEY"`
	Entrypoint                       string `env:"ENTRYPOINT"`
	Env                              string `env:"ENV" envDefault:"development"`
	JWTSecret                        string `env:"JWT_SECRET"`
}

type DatabaseEnvironmentVariable struct {
	Username     string `env:"DB_USERNAME"`
	Password     string `env:"DB_PASSWORD"`
	Host         string `env:"DB_HOST"`
	Port         string `env:"DB_PORT"`
	DatabaseName string `env:"DB_DATABASE"`
}
