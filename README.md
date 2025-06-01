# Escritorio Remoto - Backend

## 🚀 Estado del Proyecto

### ✅ FASE 1: Autenticación del Administrador - **COMPLETADA**
**Tag**: `v1.0-fase1` | **Coverage**: 92.3% | **Estado**: 100% Funcional

#### Componentes Implementados
- **Dominio**: Entidad User con validación bcrypt
- **Aplicación**: AuthService con JWT + IUserRepository interface
- **Infraestructura**: MySQLUserRepository con conexión MySQL
- **Presentación**: AuthHandler con endpoint POST /api/auth/login

#### Pruebas
- ✅ 9 pruebas unitarias AuthService (92.3% coverage)
- ✅ 6 pruebas integración MySQLUserRepository (83.3% coverage)
- ✅ Endpoint funcional probado (200/401 responses)

#### Endpoints Disponibles
- `POST /api/auth/login` - Autenticación de administradores
- `GET /health` - Health check del servidor

---

## 🏗️ Arquitectura

### Tecnologías
- **Backend**: Go 1.21+, Gin Framework
- **Base de Datos**: MySQL 8.0 + Redis 7.0 (Docker)
- **Autenticación**: JWT con bcrypt
- **Testing**: Testify con mocks

### Estructura por Capas
```
internal/
├── domain/          # Entidades y lógica de negocio
├── application/     # Casos de uso e interfaces
├── infrastructure/  # Implementaciones BD/External
└── presentation/    # Controllers HTTP y DTOs
```

### Patrones Implementados
- **Repository Pattern**: Abstracción acceso a datos
- **DTO Pattern**: Transferencia entre capas
- **Dependency Injection**: Inyección de dependencias
- **SOLID Principles**: Arquitectura limpia

---

## 🚀 Inicio Rápido

### Prerrequisitos
- Go 1.21+
- Docker & Docker Compose
- Git

### Instalación
```bash
# Clonar repositorio
git clone https://github.com/Unikyri/EscritorioRemoto-Backend.git
cd EscritorioRemoto-Backend

# Iniciar infraestructura
docker-compose up -d

# Instalar dependencias
go mod tidy

# Compilar
go build -o bin/server.exe cmd/server/main.go

# Ejecutar
./bin/server.exe
```

### Probar Autenticación
```bash
# PowerShell
$body = '{"username":"admin","password":"password"}'
Invoke-WebRequest -Uri "http://localhost:8080/api/auth/login" -Method POST -Body $body -ContentType "application/json"
```

---

## 🧪 Testing

### Ejecutar Pruebas
```bash
# Todas las pruebas
go test ./internal/... -v

# Con coverage
go test ./internal/... -cover

# Solo unitarias
go test ./internal/application/userservice/ -v -cover

# Solo integración
go test ./internal/infrastructure/database/ -v
```

### Coverage Actual
- **AuthService**: 92.3% (supera mínimo 70%)
- **MySQLUserRepository**: 83.3%
- **Total**: Cumple estándares de calidad

---

## 📋 Próximas Fases

### 🔄 FASE 2: Autenticación Usuario Cliente y Registro PC
- Autenticación usuarios cliente
- Registro de PCs cliente con servidor
- Gestión de conexiones

### 🔄 FASE 3: Visualización PCs y Estado Conexión
- Dashboard AdminWeb con lista PCs
- Estado en tiempo real
- Interfaz cliente Wails

### 🔄 FASE 4-12: Funcionalidades Avanzadas
- Control remoto con streaming
- Transferencia de archivos
- Grabación de sesiones
- Logs y auditoría
- Informes y notificaciones

---

## 📚 Documentación

- [Configuración Infraestructura](./docs/01_Configuracion_Infraestructura.md)
- [FASE 1: Autenticación Administrador](./docs/02_Fase1_Autenticacion_Administrador.md)
- [Reglas de Desarrollo](./.cursor/rules/)

---

## 🔧 Configuración

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

### Usuario Admin Inicial
- **Username**: `admin`
- **Password**: `password`
- **Role**: `ADMINISTRATOR`

---

## 🤝 Contribución

### Metodología
- **Desarrollo secuencial por fases**
- **Una fase a la vez** (no avanzar sin completar 100%)
- **Commits**: `[FASE-X] tipo: descripción`
- **Tags**: `v1.0-faseX` por cada fase completada

### Estándares de Código
- **Go**: PascalCase públicos, camelCase privados
- **Interfaces**: Prefijo `I`
- **DTOs**: Sufijo `DTO`
- **Tests**: Mínimo 70% coverage
- **Documentación**: Obligatoria para APIs públicas

---

## 📄 Licencia

Este proyecto es parte de un MVP académico para administración remota de equipos de cómputo.

---

## 🏷️ Tags y Versiones

- `v1.0-fase1` - ✅ Autenticación Administrador (COMPLETADA)
- `v1.0-fase2` - 🔄 Autenticación Cliente + Registro PC (PENDIENTE)
- `v1.0-fase3` - 🔄 Visualización PCs y Estado (PENDIENTE)

**Estado Actual**: FASE 1 100% COMPLETADA - Listo para FASE 2
