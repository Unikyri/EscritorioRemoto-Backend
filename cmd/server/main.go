package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/pcservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/database"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/handlers"
)

func main() {
	log.Println("Escritorio Remoto - Backend Server")
	log.Println("FASE 2: Autenticacion Cliente y Registro PC")

	dbConfig := database.Config{
		Host:               getEnv("DB_HOST", "localhost"),
		Port:               getEnv("DB_PORT", "3306"),
		Database:           getEnv("DB_NAME", "escritorio_remoto_db"),
		Username:           getEnv("DB_USER", "app_user"),
		Password:           getEnv("DB_PASSWORD", "app_password"),
		MaxConnections:     20,
		MaxIdleConnections: 10,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Error al conectar con la base de datos: %v", err)
	}
	defer db.Close()

	log.Println("Conexion a MySQL exitosa")

	userRepository := database.NewMySQLUserRepository(db)
	clientPCRepository := database.NewMySQLClientPCRepository(db)
	clientPCFactory := clientpc.NewClientPCFactory()

	jwtSecret := getEnv("JWT_SECRET", "escritorio_remoto_jwt_secret_development_2025")
	authService := userservice.NewAuthService(userRepository, jwtSecret)
	pcService := pcservice.NewPCService(clientPCRepository, clientPCFactory)

	authHandler := handlers.NewAuthHandler(authService)
	webSocketHandler := handlers.NewWebSocketHandler(authService, pcService)

	router := gin.Default()

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

	api := router.Group("/api")
	authHandler.RegisterRoutes(api)

	ws := router.Group("/ws")
	{
		ws.GET("/client", webSocketHandler.HandleWebSocket)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Escritorio Remoto Backend - FASE 2",
			"version": "0.2.0-fase2",
		})
	})

	port := getEnv("SERVER_PORT", "8080")
	log.Printf("Servidor iniciando en puerto %s", port)
	log.Printf("WebSocket Cliente: ws://localhost:%s/ws/client", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
