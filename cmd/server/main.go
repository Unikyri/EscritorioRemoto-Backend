package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/pcservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/remotesessionservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/userservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/clientpc"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/events"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/database"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/persistence/mysql"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/handlers"
	httpHandlers "github.com/unikyri/escritorio-remoto-backend/internal/presentation/http/handlers"
	"github.com/unikyri/escritorio-remoto-backend/internal/presentation/middleware"
)

func main() {
	log.Println("Escritorio Remoto - Backend Server")
	log.Println("FASE 4 - PASO 1: Inicio, Aceptación/Rechazo de Sesión de Control Remoto")

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

	// Inicializar dependencias para sesiones remotas
	eventBus := events.NewSimpleEventBus()
	remoteSessionRepository := mysql.NewRemoteSessionRepository(db)
	remoteSessionService := remotesessionservice.NewRemoteSessionService(
		remoteSessionRepository,
		userRepository,
		clientPCRepository,
		eventBus,
	)

	// Crear handlers con las dependencias correctas
	authHandler := handlers.NewAuthHandler(authService)
	adminWSHandler := handlers.NewAdminWebSocketHandler(authService, remoteSessionService)
	webSocketHandler := handlers.NewWebSocketHandler(authService, pcService, remoteSessionService, adminWSHandler)

	// Establecer referencia circular entre handlers
	adminWSHandler.SetClientWSHandler(webSocketHandler)

	// Configurar callback para notificar sesiones terminadas
	remoteSessionService.SetSessionEndedNotifier(func(sessionID, clientPCID, adminUserID string) {
		err := adminWSHandler.NotifySessionEnded(sessionID, clientPCID, adminUserID)
		if err != nil {
			log.Printf("Error notifying session ended: %v", err)
		}
	})

	// Configurar notificación al cliente cuando termina la sesión
	remoteSessionService.SetClientSessionEndedNotifier(func(sessionID, clientPCID string) {
		err := webSocketHandler.SendSessionEndedToClient(sessionID, clientPCID)
		if err != nil {
			log.Printf("Error notifying client session ended: %v", err)
		}
	})

	pcHandler := handlers.NewPCHandler(pcService, authService)

	// Crear handler de control remoto con WebSocket handler (no el hub separado)
	remoteControlHandler := httpHandlers.NewRemoteControlHandler(remoteSessionService, webSocketHandler)

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

		// Rutas para sesiones de control remoto
		admin.POST("/sessions/initiate", remoteControlHandler.InitiateSession)
		admin.GET("/sessions/:sessionId/status", remoteControlHandler.GetSessionStatus)
		admin.POST("/sessions/:sessionId/end", remoteControlHandler.EndSession)
		admin.GET("/sessions/active", remoteControlHandler.GetActiveSessions)
		admin.GET("/sessions/my", remoteControlHandler.GetUserSessions)
	}

	ws := router.Group("/ws")
	{
		ws.GET("/client", webSocketHandler.HandleWebSocket)
		ws.GET("/admin", adminWSHandler.HandleAdminWebSocket)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Escritorio Remoto Backend - FASE 4 PASO 1: Sesiones de Control Remoto",
			"version": "0.4.1-fase4-paso1-remote-sessions",
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
	log.Printf("API Iniciar Sesión: http://localhost:%s/api/admin/sessions/initiate", port)
	log.Printf("API Estado Sesión: http://localhost:%s/api/admin/sessions/:sessionId/status", port)
	log.Printf("API Sesiones Activas: http://localhost:%s/api/admin/sessions/active", port)
	log.Printf("API Mis Sesiones: http://localhost:%s/api/admin/sessions/my", port)

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
