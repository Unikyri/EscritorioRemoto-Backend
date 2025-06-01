package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/database"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/handlers"
)

func main() {
	log.Println("🚀 Escritorio Remoto - Backend Server")
	log.Println("📋 FASE 1: Autenticación del Administrador")

	// Configuración de la base de datos
	dbConfig := database.Config{
		Host:               getEnv("DB_HOST", "localhost"),
		Port:               getEnv("DB_PORT", "3306"),
		Database:           getEnv("DB_NAME", "escritorio_remoto_db"),
		Username:           getEnv("DB_USER", "app_user"),
		Password:           getEnv("DB_PASSWORD", "app_password"),
		MaxConnections:     20,
		MaxIdleConnections: 10,
	}

	// Conectar a la base de datos
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("❌ Error al conectar con la base de datos: %v", err)
	}
	defer db.Close()

	log.Println("✅ Conexión a MySQL exitosa")

	// Inicializar repositorio
	userRepository := database.NewMySQLUserRepository(db)

	// Inicializar servicios
	jwtSecret := getEnv("JWT_SECRET", "escritorio_remoto_jwt_secret_development_2025")
	authService := userservice.NewAuthService(userRepository, jwtSecret)

	// Inicializar handlers
	authHandler := handlers.NewAuthHandler(authService)

	// Configurar servidor HTTP
	router := gin.Default()

	// Middleware CORS básico
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Registrar rutas
	api := router.Group("/api")
	authHandler.RegisterRoutes(api)

	// Ruta de prueba
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Escritorio Remoto Backend - FASE 1",
			"version": "0.1.0-fase1",
		})
	})

	// Iniciar servidor
	port := getEnv("SERVER_PORT", "8080")
	log.Printf("🌐 Servidor iniciando en puerto %s", port)
	log.Printf("🔗 Endpoint de autenticación: http://localhost:%s/api/auth/login", port)
	log.Printf("🔗 Health check: http://localhost:%s/health", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("❌ Error al iniciar el servidor: %v", err)
	}
}

// getEnv obtiene una variable de entorno con valor por defecto
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
