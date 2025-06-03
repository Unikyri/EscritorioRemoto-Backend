package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("🔧 Probando conectividad con Redis...")

	// Configuración de conexión
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // Sin password en desarrollo
		DB:       0,  // Base de datos por defecto
	})
	defer rdb.Close()

	ctx := context.Background()

	// Verificar que la conexión funciona
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Error al conectar con Redis: %v", err)
	}

	fmt.Printf("✅ Conexión con Redis exitosa: %s\n", pong)

	// Probar operaciones básicas
	testBasicOperations(rdb, ctx)

	// Probar operaciones de caché típicas del proyecto
	testCacheOperations(rdb, ctx)

	// Probar persistencia
	testPersistence(rdb, ctx)

	fmt.Println("🎉 Todas las pruebas de Redis han pasado exitosamente!")
}

func testBasicOperations(rdb *redis.Client, ctx context.Context) {
	fmt.Println("\n📋 Probando operaciones básicas...")

	// SET y GET
	testKey := "test:basic"
	testValue := "Hello Redis!"

	err := rdb.Set(ctx, testKey, testValue, 0).Err()
	if err != nil {
		log.Fatalf("❌ Error al hacer SET: %v", err)
	}

	val, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		log.Fatalf("❌ Error al hacer GET: %v", err)
	}

	if val != testValue {
		log.Fatalf("❌ Los valores no coinciden: esperado %s, obtenido %s", testValue, val)
	}

	fmt.Printf("✅ SET/GET exitoso: %s\n", val)

	// Limpiar
	rdb.Del(ctx, testKey)
}

func testCacheOperations(rdb *redis.Client, ctx context.Context) {
	fmt.Println("\n📋 Probando operaciones de caché del proyecto...")

	// Simular caché de metadatos de PC
	pcCacheKey := "pc:status:test-pc-123"
	pcStatus := `{"pc_id":"test-pc-123","status":"ONLINE","last_seen":"2025-01-06T01:00:00Z"}`

	// Guardar con TTL (Time To Live)
	err := rdb.Set(ctx, pcCacheKey, pcStatus, 5*time.Minute).Err()
	if err != nil {
		log.Fatalf("❌ Error al cachear estado de PC: %v", err)
	}

	// Verificar TTL
	ttl, err := rdb.TTL(ctx, pcCacheKey).Result()
	if err != nil {
		log.Fatalf("❌ Error al obtener TTL: %v", err)
	}

	fmt.Printf("✅ Estado de PC cacheado con TTL: %v\n", ttl)

	// Simular caché de sesión
	sessionKey := "session:active:admin-123"
	sessionData := `{"session_id":"sess-456","admin_id":"admin-123","pc_id":"test-pc-123","started_at":"2025-01-06T01:00:00Z"}`

	err = rdb.Set(ctx, sessionKey, sessionData, 1*time.Hour).Err()
	if err != nil {
		log.Fatalf("❌ Error al cachear sesión: %v", err)
	}

	// Verificar que se puede leer
	cachedSession, err := rdb.Get(ctx, sessionKey).Result()
	if err != nil {
		log.Fatalf("❌ Error al leer sesión cacheada: %v", err)
	}

	fmt.Printf("✅ Sesión cacheada correctamente: %s\n", cachedSession[:50]+"...")

	// Probar operaciones de lista (para logs en tiempo real)
	logKey := "logs:realtime"
	logEntry := fmt.Sprintf(`{"timestamp":"%s","action":"TEST_LOG","user":"system"}`, time.Now().Format(time.RFC3339))

	err = rdb.LPush(ctx, logKey, logEntry).Err()
	if err != nil {
		log.Fatalf("❌ Error al agregar log: %v", err)
	}

	// Obtener logs recientes (últimos 5)
	logs, err := rdb.LRange(ctx, logKey, 0, 4).Result()
	if err != nil {
		log.Fatalf("❌ Error al obtener logs: %v", err)
	}

	fmt.Printf("✅ Logs en tiempo real: %d entradas\n", len(logs))

	// Limpiar datos de prueba
	rdb.Del(ctx, pcCacheKey, sessionKey, logKey)
}

func testPersistence(rdb *redis.Client, ctx context.Context) {
	fmt.Println("\n📋 Probando persistencia...")

	// Crear datos que deberían persistir
	persistKey := "test:persistence"
	persistValue := fmt.Sprintf("Datos creados en: %s", time.Now().Format(time.RFC3339))

	err := rdb.Set(ctx, persistKey, persistValue, 0).Err() // Sin TTL
	if err != nil {
		log.Fatalf("❌ Error al crear datos persistentes: %v", err)
	}

	fmt.Printf("✅ Datos persistentes creados: %s\n", persistValue)

	// Verificar información del servidor Redis
	info, err := rdb.Info(ctx, "persistence").Result()
	if err != nil {
		log.Printf("⚠️  No se pudo obtener info de persistencia: %v", err)
	} else {
		fmt.Println("✅ Información de persistencia obtenida")
		// Mostrar solo las primeras líneas para no saturar la salida
		lines := info[:min(200, len(info))]
		fmt.Printf("📊 Info Redis: %s...\n", lines)
	}

	// Verificar que los datos siguen ahí
	retrievedValue, err := rdb.Get(ctx, persistKey).Result()
	if err != nil {
		log.Fatalf("❌ Error al recuperar datos persistentes: %v", err)
	}

	if retrievedValue != persistValue {
		log.Fatalf("❌ Los datos persistentes no coinciden")
	}

	fmt.Println("✅ Persistencia verificada correctamente")

	// Limpiar
	rdb.Del(ctx, persistKey)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
