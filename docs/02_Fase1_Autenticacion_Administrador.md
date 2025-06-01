# FASE 1: Autenticación del Administrador

## Resumen
La FASE 1 implementa el sistema de autenticación para administradores del sistema, incluyendo:
- Autenticación con JWT
- Validación de credenciales
- Protección de endpoints
- Arquitectura por capas siguiendo principios SOLID

## Componentes Implementados

### 1. Dominio (Domain Layer)
**Archivo**: `internal/domain/user/user.go`

- **Entidad User**: Representa un usuario del sistema
- **Role enum**: Define roles ADMINISTRATOR y CLIENT_USER
- **Métodos de negocio**:
  - `ValidatePassword()`: Valida contraseñas usando bcrypt
  - `IsAdministrator()`: Verifica si es administrador
  - `Deactivate()`: Desactiva usuario
  - `ToSnapshot()`: Convierte a representación inmutable

### 2. Aplicación (Application Layer)
**Archivos**:
- `internal/application/interfaces/user_repository.go`
- `internal/application/userservice/auth_service.go`

#### IUserRepository Interface
```go
type IUserRepository interface {
    FindByUsername(username string) (*user.User, error)
    FindByID(userID string) (*user.User, error)
    Save(user *user.User) error
    Create(user *user.User) error
}
```

#### AuthService
- **AuthenticateAdmin()**: Autentica administradores y genera JWT
- **ValidateToken()**: Valida tokens JWT
- **Validaciones**:
  - Usuario existe
  - Es administrador
  - Está activo
  - Contraseña correcta

### 3. Infraestructura (Infrastructure Layer)
**Archivos**:
- `internal/infrastructure/database/connection.go`
- `internal/infrastructure/database/mysql_user_repository.go`

#### MySQLUserRepository
Implementa `IUserRepository` con:
- Conexión a MySQL con pool de conexiones
- Queries preparados para seguridad
- Manejo de errores robusto
- Conversión entre entidades de dominio y BD

### 4. Presentación (Presentation Layer)
**Archivos**:
- `internal/presentation/dto/auth_dto.go`
- `internal/presentation/handlers/auth_handler.go`

#### DTOs
- **AuthRequestDTO**: `{username, password}`
- **AuthResponseDTO**: `{token, user}`
- **UserInfoDTO**: Información del usuario
- **ErrorResponseDTO**: Respuestas de error estandarizadas

#### AuthHandler
- **POST /api/auth/login**: Endpoint de autenticación
- Validación de entrada con Gin binding
- Manejo de errores HTTP apropiado
- Respuestas JSON estructuradas

## Endpoints API

### POST /api/auth/login
**Request**:
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response Success (200)**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "user_id": "admin-000-000-000-000000000001",
    "username": "admin",
    "role": "ADMINISTRATOR",
    "is_active": true,
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

**Response Error (401)**:
```json
{
  "error": "authentication_failed",
  "message": "Invalid credentials or user is not an administrator",
  "code": 401
}
```

### GET /health
**Response (200)**:
```json
{
  "status": "ok",
  "message": "Escritorio Remoto Backend - FASE 1",
  "version": "0.1.0-fase1"
}
```

## Seguridad Implementada

### JWT (JSON Web Tokens)
- **Algoritmo**: HS256
- **Expiración**: 24 horas
- **Claims personalizados**:
  - `user_id`: ID del usuario
  - `username`: Nombre de usuario
  - `role`: Rol del usuario
- **Issuer**: "escritorio-remoto-backend"

### Contraseñas
- **Hashing**: bcrypt con costo por defecto
- **Validación**: Comparación segura con hash almacenado
- **No exposición**: Contraseñas nunca se retornan en APIs

### CORS
- Configurado para desarrollo con `*`
- Headers permitidos: `Content-Type, Authorization`
- Métodos: `GET, POST, PUT, DELETE, OPTIONS`

## Pruebas Implementadas

### Pruebas Unitarias
**Archivo**: `internal/application/userservice/auth_service_test.go`

**Coverage**: 92.3% (supera el mínimo del 70%)

**Casos de prueba**:
- ✅ Autenticación exitosa
- ✅ Usuario no encontrado
- ✅ Contraseña incorrecta
- ✅ Usuario no es administrador
- ✅ Usuario inactivo
- ✅ Error de repositorio
- ✅ Validación de token exitosa
- ✅ Token inválido
- ✅ Token expirado

### Pruebas de Integración
**Archivo**: `internal/infrastructure/database/mysql_user_repository_test.go`

**Casos de prueba**:
- ✅ Buscar usuario por username
- ✅ Usuario no encontrado
- ✅ Buscar usuario por ID
- ✅ Crear usuario
- ✅ Actualizar usuario
- ✅ Validación de contraseñas

### Pruebas de Endpoint
**Verificadas manualmente**:
- ✅ Health check funcional
- ✅ Login con credenciales correctas (200)
- ✅ Login con credenciales incorrectas (401)
- ✅ Formato de respuesta JSON correcto
- ✅ Headers CORS configurados

## Configuración

### Variables de Entorno
```bash
# Base de datos
DB_HOST=localhost
DB_PORT=3306
DB_NAME=escritorio_remoto_db
DB_USER=app_user
DB_PASSWORD=app_password

# JWT
JWT_SECRET=escritorio_remoto_jwt_secret_development_2025

# Servidor
SERVER_PORT=8080
```

### Usuario Administrador Inicial
- **Username**: `admin`
- **Password**: `password`
- **ID**: `admin-000-000-000-000000000001`
- **Role**: `ADMINISTRATOR`

## Arquitectura y Patrones

### Principios SOLID Aplicados
- **S**: Cada clase tiene una responsabilidad única
- **O**: Abierto para extensión, cerrado para modificación
- **L**: Las implementaciones son sustituibles
- **I**: Interfaces segregadas por funcionalidad
- **D**: Dependencias invertidas con interfaces

### Patrones Implementados
- **Repository Pattern**: Abstracción de acceso a datos
- **DTO Pattern**: Transferencia de datos entre capas
- **Factory Pattern**: Creación de entidades User
- **Dependency Injection**: Inyección manual en main.go

### Arquitectura por Capas
```
Presentation Layer (HTTP/JSON)
    ↓
Application Layer (Casos de Uso)
    ↓
Domain Layer (Lógica de Negocio)
    ↓
Infrastructure Layer (BD/External)
```

## Comandos de Ejecución

### Iniciar Infraestructura
```bash
docker-compose up -d
```

### Compilar y Ejecutar
```bash
go build -o bin/server.exe cmd/server/main.go
./bin/server.exe
```

### Ejecutar Pruebas
```bash
# Pruebas unitarias
go test ./internal/application/userservice/ -v -cover

# Pruebas de integración
go test ./internal/infrastructure/database/ -v

# Todas las pruebas
go test ./... -v
```

### Probar Endpoint
```bash
# PowerShell
$body = '{"username":"admin","password":"password"}'
Invoke-WebRequest -Uri "http://localhost:8080/api/auth/login" -Method POST -Body $body -ContentType "application/json"
```

## Estado de Completitud

### ✅ Criterios Cumplidos
- [x] Código funciona sin errores
- [x] Tests pasan al 100% (coverage 92.3% > 70%)
- [x] Documentación actualizada
- [x] Patrones de diseño implementados correctamente
- [x] Arquitectura por capas respetada
- [x] Principios SOLID aplicados
- [x] Endpoint funcional y probado
- [x] Seguridad implementada (JWT + bcrypt)
- [x] Manejo de errores robusto

### 🎯 Próximos Pasos
La FASE 1 está **100% COMPLETA** y lista para:
- Commit y tag en GitHub: `v1.0-fase1`
- Inicio de FASE 2: Autenticación Usuario Cliente y Registro del PC

## Notas Técnicas

### Decisiones de Diseño
1. **JWT vs Sessions**: JWT elegido por escalabilidad y stateless
2. **bcrypt**: Estándar de la industria para hashing de contraseñas
3. **Gin Framework**: Ligero y performante para APIs REST
4. **Testify**: Framework de testing robusto con mocks
5. **Arquitectura por capas**: Separación clara de responsabilidades

### Consideraciones de Producción
- JWT secret debe ser más robusto en producción
- CORS debe configurarse específicamente por dominio
- Logs de auditoría para intentos de login
- Rate limiting para prevenir ataques de fuerza bruta
- HTTPS obligatorio en producción 