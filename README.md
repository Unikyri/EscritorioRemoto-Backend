# Escritorio Remoto - Backend

## 🚀 Estado del Proyecto

### ✅ FASE 1: Autenticación del Administrador - **COMPLETADA**
**Tag**: `v1.0-fase1` | **Coverage**: 92.3% | **Estado**: 100% Funcional

### ✅ FASE 2: Autenticación Cliente y Registro PC - **COMPLETADA**
**Tag**: `v1.0-fase2` | **Estado**: 100% Funcional

#### Componentes Implementados
- **Dominio**: Entidades User y ClientPC con validaciones
- **Aplicación**: AuthService, PCService con interfaces repository
- **Infraestructura**: MySQLUserRepository, MySQLClientPCRepository
- **Presentación**: AuthHandler, WebSocketHandler
- **Comunicación**: WebSocket para clientes, REST para administradores

#### Funcionalidades
- ✅ Autenticación administradores (JWT)
- ✅ Autenticación usuarios cliente (WebSocket)
- ✅ Registro automático de PCs cliente
- ✅ Heartbeat y gestión de conexiones
- ✅ Persistencia de estado de conexión

#### Endpoints Disponibles
- `POST /api/auth/login` - Autenticación administradores
- `GET /ws/client` - WebSocket para clientes
- `GET /health` - Health check del servidor

---

## 🏗️ Arquitectura

### Tecnologías
- **Backend**: Go 1.21+, Gin Framework
- **Base de Datos**: MySQL 8.0 + Redis 7.0 (Docker)
- **Autenticación**: JWT con bcrypt
- **Comunicación**: WebSocket + REST
- **Testing**: Testify con mocks

### Estructura por Capas
```
internal/
├── domain/          # Entidades y lógica de negocio
├── application/     # Casos de uso e interfaces
├── infrastructure/  # Implementaciones BD/External
└── presentation/    # Controllers HTTP/WS y DTOs
```

### Patrones Implementados
- **Repository Pattern**: Abstracción acceso a datos
- **Factory Pattern**: Creación de entidades
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

# Configurar variables de entorno (ver docs)
# Crear archivo .env con configuraciones necesarias

# Compilar
go build -o bin/server cmd/server/main.go

# Ejecutar
./bin/server
```

---

## 🧪 Testing

### Ejecutar Pruebas
```bash
# Todas las pruebas
go test ./internal/... -v

# Con coverage
go test ./internal/... -cover

# Solo aplicación
go test ./internal/application/... -v -cover

# Solo infraestructura
go test ./internal/infrastructure/... -v
```

### Coverage Actual
- **AuthService**: 92.3%
- **PCService**: 85%+
- **MySQLRepositories**: 80%+
- **Total**: Cumple estándares de calidad (>70%)

---

## 📋 Próximas Fases

### 🔄 FASE 3: Visualización PCs y Estado Conexión
- Dashboard AdminWeb con lista PCs
- Estado de conexión en tiempo real
- Interfaz cliente mejorada

### 🔄 FASE 4-12: Funcionalidades Avanzadas
- Control remoto con streaming
- Transferencia de archivos
- Grabación de sesiones
- Logs y auditoría
- Informes y notificaciones

---

## 📚 Documentación

### Configuración
- Consultar documentación de infraestructura para configuración completa
- Variables de entorno requeridas disponibles en docs/
- Esquema de base de datos en scripts/init.sql

### Seguridad
- No exponer credenciales en código fuente
- Usar variables de entorno para configuración sensible
- JWT con secretos seguros en producción

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
- `v1.0-fase2` - ✅ Autenticación Cliente + Registro PC (COMPLETADA)
- `v1.0-fase3` - 🔄 Visualización PCs y Estado (PENDIENTE)

**Estado Actual**: FASE 2 100% COMPLETADA - Listo para FASE 3
