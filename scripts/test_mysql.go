package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("🔧 Probando conectividad con MySQL...")

	// Configuración de conexión
	dsn := "app_user:app_password@tcp(localhost:3306)/escritorio_remoto_db?parseTime=true"

	// Intentar conexión
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ Error al conectar con MySQL: %v", err)
	}
	defer db.Close()

	// Verificar que la conexión funciona
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Error al hacer ping a MySQL: %v", err)
	}

	fmt.Println("✅ Conexión con MySQL exitosa")

	// Probar que el usuario admin existe
	testAdminUser(db)

	// Probar inserción de un usuario cliente de prueba
	testClientUserInsertion(db)

	// Probar logs
	testActionLogs(db)

	fmt.Println("🎉 Todas las pruebas de MySQL han pasado exitosamente!")
}

func testAdminUser(db *sql.DB) {
	fmt.Println("\n📋 Probando usuario administrador...")

	var userID, username, role string
	var isActive bool

	query := "SELECT user_id, username, role, is_active FROM users WHERE role = 'ADMINISTRATOR' LIMIT 1"
	err := db.QueryRow(query).Scan(&userID, &username, &role, &isActive)

	if err != nil {
		log.Fatalf("❌ Error al obtener usuario administrador: %v", err)
	}

	fmt.Printf("✅ Usuario administrador encontrado: %s (ID: %s, Activo: %v)\n", username, userID, isActive)
}

func testClientUserInsertion(db *sql.DB) {
	fmt.Println("\n📋 Probando inserción de usuario cliente...")

	// Generar datos de prueba
	testUserID := uuid.New().String()
	testUsername := fmt.Sprintf("test_user_%d", time.Now().Unix())
	testPassword := "test_password_123"

	// Hash de la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), 10)
	if err != nil {
		log.Fatalf("❌ Error al generar hash de contraseña: %v", err)
	}

	// Insertar usuario de prueba
	insertQuery := `
		INSERT INTO users (user_id, username, ip, hashed_password, role, is_active) 
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, testUserID, testUsername, "192.168.1.100", string(hashedPassword), "CLIENT_USER", true)
	if err != nil {
		log.Fatalf("❌ Error al insertar usuario de prueba: %v", err)
	}

	fmt.Printf("✅ Usuario cliente de prueba insertado: %s (ID: %s)\n", testUsername, testUserID)

	// Verificar que se puede leer
	var retrievedUsername string
	selectQuery := "SELECT username FROM users WHERE user_id = ?"
	err = db.QueryRow(selectQuery, testUserID).Scan(&retrievedUsername)
	if err != nil {
		log.Fatalf("❌ Error al leer usuario insertado: %v", err)
	}

	if retrievedUsername != testUsername {
		log.Fatalf("❌ Los datos no coinciden: esperado %s, obtenido %s", testUsername, retrievedUsername)
	}

	fmt.Printf("✅ Persistencia verificada: usuario %s leído correctamente\n", retrievedUsername)

	// Limpiar datos de prueba
	deleteQuery := "DELETE FROM users WHERE user_id = ?"
	_, err = db.Exec(deleteQuery, testUserID)
	if err != nil {
		log.Printf("⚠️  Advertencia: no se pudo limpiar el usuario de prueba: %v", err)
	} else {
		fmt.Println("✅ Usuario de prueba eliminado correctamente")
	}
}

func testActionLogs(db *sql.DB) {
	fmt.Println("\n📋 Probando logs de acciones...")

	// Verificar que existen logs
	var logCount int
	countQuery := "SELECT COUNT(*) FROM action_logs"
	err := db.QueryRow(countQuery).Scan(&logCount)
	if err != nil {
		log.Fatalf("❌ Error al contar logs: %v", err)
	}

	fmt.Printf("✅ Total de logs en el sistema: %d\n", logCount)

	// Obtener el log más reciente
	var logID int64
	var actionType, description string
	var timestamp time.Time

	recentLogQuery := `
		SELECT log_id, action_type, description, timestamp 
		FROM action_logs 
		ORDER BY timestamp DESC 
		LIMIT 1
	`

	err = db.QueryRow(recentLogQuery).Scan(&logID, &actionType, &description, &timestamp)
	if err != nil {
		log.Fatalf("❌ Error al obtener log más reciente: %v", err)
	}

	fmt.Printf("✅ Log más reciente: ID=%d, Tipo=%s, Fecha=%v\n", logID, actionType, timestamp.Format("2006-01-02 15:04:05"))
}
