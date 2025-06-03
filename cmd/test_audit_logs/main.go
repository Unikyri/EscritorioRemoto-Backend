package main

import (
	"context"
	"log"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/actionlogservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/database"
	"github.com/unikyri/escritorio-remoto-backend/internal/infrastructure/persistence/mysql"
)

func main() {
	log.Println("🔍 Testing ActionLog Implementation...")

	// Configurar conexión a base de datos
	dbConfig := database.Config{
		Host:               "localhost",
		Port:               "3306",
		Database:           "escritorio_remoto_db",
		Username:           "app_user",
		Password:           "app_password",
		MaxConnections:     20,
		MaxIdleConnections: 10,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("❌ Error connecting to database: %v", err)
	}
	defer db.Close()

	log.Println("✅ Connected to database")

	// Crear repositorio y servicio
	actionLogRepo := mysql.NewActionLogRepository(db)
	actionLogService := actionlogservice.NewActionLogService(actionLogRepo)

	ctx := context.Background()

	// 1. Verificar logs existentes
	log.Println("\n📋 Checking existing logs...")
	recentLogs, err := actionLogService.GetRecentLogs(ctx, 5)
	if err != nil {
		log.Printf("⚠️ Error getting recent logs: %v", err)
	} else {
		log.Printf("✅ Found %d recent logs", len(recentLogs))
		for i, actionLog := range recentLogs {
			log.Printf("  [%d] %s: %s (by %s)", 
				i+1, 
				actionLog.ActionType(), 
				actionLog.Description(), 
				actionLog.PerformedByUserID())
		}
	}

	// 2. Crear un log de prueba
	log.Println("\n📝 Creating test audit log...")
	testSessionID := "test-session-12345"
	testAdminID := "admin-000-000-000-000000000001"
	
	err = actionLogService.LogSessionEnded(ctx, testSessionID, testAdminID, "test_ended_by_admin")
	if err != nil {
		log.Printf("❌ Error creating test log: %v", err)
		return
	}
	
	log.Println("✅ Test audit log created successfully!")

	// 3. Verificar que se guardó
	log.Println("\n🔍 Verifying test log was saved...")
	time.Sleep(100 * time.Millisecond) // Pequeña pausa

	recentLogs, err = actionLogService.GetRecentLogs(ctx, 1)
	if err != nil {
		log.Printf("❌ Error getting recent logs after test: %v", err)
		return
	}

	if len(recentLogs) > 0 {
		latestLog := recentLogs[0]
		log.Printf("✅ Latest log found:")
		log.Printf("   Type: %s", latestLog.ActionType())
		log.Printf("   Description: %s", latestLog.Description())
		log.Printf("   Performed by: %s", latestLog.PerformedByUserID())
		log.Printf("   Timestamp: %s", latestLog.Timestamp().Format("2006-01-02 15:04:05"))
		
		if latestLog.SubjectEntityID() != nil {
			log.Printf("   Subject Entity: %s", *latestLog.SubjectEntityID())
		}
		if latestLog.SubjectEntityType() != nil {
			log.Printf("   Entity Type: %s", *latestLog.SubjectEntityType())
		}
		
		if latestLog.Details() != nil {
			log.Printf("   Details: %+v", latestLog.Details())
		}
	}

	// 4. Obtener total de logs
	totalLogs, err := actionLogService.GetLogsCount(ctx)
	if err != nil {
		log.Printf("⚠️ Error getting total logs count: %v", err)
	} else {
		log.Printf("\n📊 Total audit logs in database: %d", totalLogs)
	}

	log.Println("\n🎉 ActionLog test completed!")
} 