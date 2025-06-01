package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// AuthRequest estructura para la petición de login
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse estructura para la respuesta de login
type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		UserID    string    `json:"user_id"`
		Username  string    `json:"username"`
		Role      string    `json:"role"`
		IsActive  bool      `json:"is_active"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"user"`
}

// ErrorResponse estructura para respuestas de error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func main() {
	fmt.Println("🧪 Probando endpoint de autenticación...")

	baseURL := "http://localhost:8080"

	// Probar health check primero
	testHealthCheck(baseURL)

	// Probar login con credenciales incorrectas
	testInvalidCredentials(baseURL)

	// Probar login con credenciales correctas del admin
	testValidAdminCredentials(baseURL)

	fmt.Println("🎉 Todas las pruebas del endpoint de autenticación completadas!")
}

func testHealthCheck(baseURL string) {
	fmt.Println("\n📋 Probando health check...")

	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		log.Printf("❌ Error en health check: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ Health check exitoso (Status: %d)\n", resp.StatusCode)
	} else {
		fmt.Printf("⚠️  Health check con status inesperado: %d\n", resp.StatusCode)
	}
}

func testInvalidCredentials(baseURL string) {
	fmt.Println("\n📋 Probando credenciales incorrectas...")

	request := AuthRequest{
		Username: "admin",
		Password: "wrong_password",
	}

	response, statusCode := makeAuthRequest(baseURL, request)

	if statusCode == http.StatusUnauthorized {
		fmt.Printf("✅ Credenciales incorrectas rechazadas correctamente (Status: %d)\n", statusCode)

		var errorResp ErrorResponse
		if err := json.Unmarshal(response, &errorResp); err == nil {
			fmt.Printf("   Error: %s\n", errorResp.Error)
			fmt.Printf("   Mensaje: %s\n", errorResp.Message)
		}
	} else {
		fmt.Printf("❌ Respuesta inesperada para credenciales incorrectas (Status: %d)\n", statusCode)
	}
}

func testValidAdminCredentials(baseURL string) {
	fmt.Println("\n📋 Probando credenciales correctas del admin...")

	request := AuthRequest{
		Username: "admin",
		Password: "password", // Contraseña del usuario admin inicial
	}

	response, statusCode := makeAuthRequest(baseURL, request)

	if statusCode == http.StatusOK {
		fmt.Printf("✅ Autenticación exitosa (Status: %d)\n", statusCode)

		var authResp AuthResponse
		if err := json.Unmarshal(response, &authResp); err == nil {
			fmt.Printf("   Token JWT recibido: %s...\n", authResp.Token[:20])
			fmt.Printf("   Usuario: %s (ID: %s)\n", authResp.User.Username, authResp.User.UserID)
			fmt.Printf("   Rol: %s\n", authResp.User.Role)
			fmt.Printf("   Activo: %v\n", authResp.User.IsActive)
		} else {
			fmt.Printf("❌ Error al parsear respuesta exitosa: %v\n", err)
		}
	} else {
		fmt.Printf("❌ Error en autenticación válida (Status: %d)\n", statusCode)
		fmt.Printf("   Respuesta: %s\n", string(response))
	}
}

func makeAuthRequest(baseURL string, request AuthRequest) ([]byte, int) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Printf("❌ Error al crear JSON: %v", err)
		return nil, 0
	}

	resp, err := http.Post(baseURL+"/api/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Error en petición HTTP: %v", err)
		return nil, 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Error al leer respuesta: %v", err)
		return nil, resp.StatusCode
	}

	return body, resp.StatusCode
}
