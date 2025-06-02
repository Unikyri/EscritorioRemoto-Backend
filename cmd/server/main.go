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
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/middleware"
)

func main() {
	log.Println("Escritorio Remoto - Backend Server")
	log.Println("FASE 3 - PASO 1: Visualización de PCs Cliente y Estado")

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
	adminWSHandler := handlers.NewAdminWebSocketHandler(authService)
	webSocketHandler := handlers.NewWebSocketHandler(authService, pcService, adminWSHandler)
	pcHandler := handlers.NewPCHandler(pcService, authService)

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

	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(authService))
	{
		admin.GET("/pcs", pcHandler.GetAllClientPCs)
		admin.GET("/pcs/online", pcHandler.GetOnlineClientPCs)
	}

	ws := router.Group("/ws")
	{
		ws.GET("/client", webSocketHandler.HandleWebSocket)
		ws.GET("/admin", middleware.AuthMiddleware(authService), adminWSHandler.HandleAdminWebSocket)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Escritorio Remoto Backend - FASE 3 PASO 1 + Notificaciones",
			"version": "0.3.1-fase3-paso1-notifications",
		})
	})

	// Endpoint temporal de debug sin autenticación
	router.GET("/debug/pcs", func(c *gin.Context) {
		log.Printf("DEBUG /debug/pcs: Starting query")

		// Intentar query sin límites
		pcs, err := clientPCRepository.FindAll(c.Request.Context(), 0, 0)
		if err != nil {
			log.Printf("DEBUG /debug/pcs: Database error: %v", err)
			c.JSON(500, gin.H{
				"error":   "Database error",
				"message": err.Error(),
			})
			return
		}

		log.Printf("DEBUG /debug/pcs: Query successful, count=%d", len(pcs))

		// Log cada PC encontrado
		for i, pc := range pcs {
			log.Printf("DEBUG /debug/pcs: PC[%d] = ID:%s, Identifier:%s, Status:%s, Owner:%s",
				i, pc.PCID, pc.Identifier, pc.ConnectionStatus, pc.OwnerUserID)
		}

		c.JSON(200, gin.H{
			"success": true,
			"count":   len(pcs),
			"data":    pcs,
			"message": "Debug endpoint - PCs retrieved without authentication",
		})
	})

	port := getEnv("SERVER_PORT", "8080")
	log.Printf("Servidor iniciando en puerto %s", port)
	log.Printf("WebSocket Cliente: ws://localhost:%s/ws/client", port)
	log.Printf("WebSocket Admin: ws://localhost:%s/ws/admin", port)
	log.Printf("API Admin PCs: http://localhost:%s/api/admin/pcs", port)
	log.Printf("API Admin PCs Online: http://localhost:%s/api/admin/pcs/online", port)

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
