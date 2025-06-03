# FASE 5 - PASO 2: Enrutamiento de Mensajes WebSocket (Backend)

## Resumen de Implementación

Este documento describe la implementación del **Paso 2 de la Fase 5** del proyecto de escritorio remoto, que se enfoca en el **enrutamiento correcto de mensajes WebSocket** entre clientes Wails y administradores web a través del servidor backend.

## Arquitectura Implementada

### Flujo de Mensajes

#### 1. Screen Frames (Cliente → Administrador)
```
Cliente Wails → WebSocketHandler → RemoteSessionService (validación) → AdminWebSocketHandler → AdminWeb
```

#### 2. Input Commands (Administrador → Cliente)  
```
AdminWeb → AdminWebSocketHandler → RemoteSessionService (validación) → WebSocketHandler → Cliente Wails
```

### Componentes Modificados

#### 1. DTOs de Mensajes WebSocket (`presentation/dto/websocket_messages.go`)

**Nuevos Tipos de Mensaje:**
```go
const (
    MessageTypeScreenFrame   = "screen_frame"
    MessageTypeInputCommand  = "input_command"
)
```

**Estructuras Añadidas:**
```go
// ScreenFrame - Frame de pantalla capturado del cliente
type ScreenFrame struct {
    SessionID   string `json:"session_id"`
    Timestamp   int64  `json:"timestamp"`
    Width       int    `json:"width"`
    Height      int    `json:"height"`
    Format      string `json:"format"`      // "jpeg", "png", etc.
    Quality     int    `json:"quality,omitempty"` // Compresión JPEG (1-100)
    FrameData   []byte `json:"frame_data"`  // Bytes de imagen
    SequenceNum int64  `json:"sequence_num"`
}

// InputCommand - Comando de input remoto del administrador
type InputCommand struct {
    SessionID   string                 `json:"session_id"`
    Timestamp   int64                  `json:"timestamp"`
    EventType   string                 `json:"event_type"` // "mouse", "keyboard"
    Action      string                 `json:"action"`     // "move", "click", "scroll", "keydown", "keyup", "type"
    Payload     map[string]interface{} `json:"payload"`    // Datos específicos del evento
}
```

#### 2. RemoteSessionService (`application/remotesessionservice/remote_session_service.go`)

**Métodos de Validación Añadidos:**

- `IsSessionActiveForStreaming(sessionID)` - Verifica si sesión está activa para streaming
- `GetActiveSessionForPC(clientPCID)` - Obtiene sesión activa para un PC específico  
- `GetAdminUserIDForActiveSession(sessionID)` - ID del admin para sesión activa
- `GetClientPCIDForActiveSession(sessionID)` - ID del PC cliente para sesión activa
- `ValidateStreamingPermission(sessionID, clientPCID)` - Valida permisos de streaming
- `ValidateInputCommandPermission(sessionID, adminUserID)` - Valida permisos de comandos

**Validación de Seguridad:**
- Solo sesiones en estado `ACTIVE` pueden procesar mensajes
- Verificación de coincidencia de PC cliente para frames
- Verificación de coincidencia de administrador para comandos

#### 3. WebSocketHandler (`presentation/handlers/websocket_handler.go`)

**Modificaciones Estructurales:**
```go
type WebSocketHandler struct {
    authService       *userservice.AuthService
    pcService         pcservice.IPCService
    sessionService    *remotesessionservice.RemoteSessionService // NUEVO
    adminWSHandler    *AdminWebSocketHandler
    connections       map[string]*ClientConnection
    pcConnections     map[string]*ClientConnection
    mutex             sync.RWMutex
}
```

**Métodos Añadidos:**

- `handleScreenFrame()` - Procesa frames de pantalla de clientes
  - Valida autenticación y registro del cliente
  - Parsea datos del frame
  - Valida permisos de streaming con `ValidateStreamingPermission()`
  - Obtiene administrador objetivo con `GetAdminUserIDForActiveSession()`
  - Reenvía frame al admin via `ForwardScreenFrameToAdmin()`

- `SendInputCommandToClient()` - Envía comandos de input a clientes
  - Encuentra conexión del cliente por PC ID
  - Verifica autenticación del cliente
  - Envía comando via WebSocket JSON

**Manejo de Mensajes Extendido:**
```go
case dto.MessageTypeScreenFrame:
    h.handleScreenFrame(conn, clientConn, message.Data)
```

#### 4. AdminWebSocketHandler (`presentation/handlers/admin_websocket_handler.go`)

**Modificaciones Estructurales:**
```go
type AdminWebSocketHandler struct {
    authService      *userservice.AuthService
    sessionService   *remotesessionservice.RemoteSessionService // NUEVO
    clientWSHandler  *WebSocketHandler // Referencia circular
    upgrader         websocket.Upgrader
    adminConnections map[string]*AdminConnection
    mutex            sync.RWMutex
}
```

**Métodos Añadidos:**

- `SetClientWSHandler()` - Establece referencia circular para evitar dependencias
- `handleInputCommand()` - Procesa comandos de input de administradores
  - Parsea comando de input
  - Valida permisos con `ValidateInputCommandPermission()`
  - Obtiene PC cliente objetivo con `GetClientPCIDForActiveSession()`
  - Reenvía comando via `SendInputCommandToClient()`

- `ForwardScreenFrameToAdmin()` - Reenvía frames a administrador específico
  - Encuentra conexión del administrador por User ID
  - Envía frame via WebSocket JSON

- `SendInputCommandToClientByAdmin()` - Método alternativo para envío de comandos

**Manejo de Mensajes Extendido:**
```go
case dto.MessageTypeInputCommand:
    h.handleInputCommand(adminConn, message.Data)
```

#### 5. Inicialización del Sistema (`cmd/server/main.go`)

**Orden de Inicialización Corregido:**
```go
// 1. Crear servicios base
remoteSessionService := remotesessionservice.NewRemoteSessionService(...)

// 2. Crear handlers con dependencias
adminWSHandler := handlers.NewAdminWebSocketHandler(authService, remoteSessionService)
webSocketHandler := handlers.NewWebSocketHandler(authService, pcService, remoteSessionService, adminWSHandler)

// 3. Establecer referencia circular
adminWSHandler.SetClientWSHandler(webSocketHandler)
```

## Características de Seguridad

### Validación de Sesiones
- **Solo sesiones ACTIVE** pueden procesar mensajes
- Verificación de coincidencia de sesión con PC/Admin
- Autenticación requerida para todos los participantes

### Validación de Mensajes
- Parsing seguro de JSON con manejo de errores
- Verificación de tipos de mensaje válidos
- Logging detallado para auditoría

### Control de Acceso
- Screen frames solo de PCs registrados y autenticados
- Input commands solo de administradores autenticados
- Validación cruzada de permisos por sesión

## Protocolo de Mensajes

### Screen Frame (Cliente → Admin)
```json
{
    "type": "screen_frame",
    "data": {
        "session_id": "session-uuid",
        "timestamp": 1640995200000,
        "width": 1920,
        "height": 1080,
        "format": "jpeg",
        "quality": 75,
        "frame_data": [bytes...],
        "sequence_num": 123
    }
}
```

### Input Command (Admin → Cliente)
```json
{
    "type": "input_command",
    "data": {
        "session_id": "session-uuid",
        "timestamp": 1640995200000,
        "event_type": "mouse",
        "action": "click",
        "payload": {
            "x": 100,
            "y": 200,
            "button": "left"
        }
    }
}
```

## Logging y Debugging

### Formato de Logs
- **📹 SCREEN FRAME**: Eventos de frames de pantalla
- **🎮 INPUT COMMAND**: Eventos de comandos de input  
- **✅**: Operaciones exitosas
- **❌**: Errores y validaciones fallidas
- **⚠️**: Advertencias y condiciones especiales

### Información Registrada
- IDs de sesión y participantes
- Secuencia de frames y tipos de comandos
- Tiempos de procesamiento y errores
- Estados de conexión y autenticación

## Performance y Optimización

### Manejo de Conexiones
- **Thread-safe** con `sync.RWMutex`
- Mapas eficientes por PC ID y User ID
- Limpieza automática de conexiones cerradas

### Procesamiento de Frames
- Envío asíncrono sin bloqueo de captura
- Validación temprana para evitar procesamiento innecesario
- Logging optimizado para debugging

### Gestión de Memoria
- Parsing directo de JSON sin copias intermedias
- Referencias controladas para evitar memory leaks
- Limpieza automática en desconexiones

## Testing y Validación

### Casos de Prueba Recomendados

1. **Screen Frame Routing**
   - Cliente envía frame → Admin lo recibe
   - Múltiples admins → Solo target recibe
   - Sesión inactiva → Frame rechazado

2. **Input Command Routing**
   - Admin envía comando → Cliente lo recibe
   - Múltiples clientes → Solo target recibe  
   - Sesión inactiva → Comando rechazado

3. **Validación de Seguridad**
   - Cliente no autenticado → Frames rechazados
   - Admin sin permisos → Comandos rechazados
   - PC/Admin incorrecto → Validación falla

4. **Manejo de Errores**
   - Conexiones cerradas inesperadamente
   - JSON malformado
   - Referencias circulares

## Próximos Pasos (Fase 5 - Paso 3)

### Integración con Frontend Web Admin
1. **Visualización de Frames**
   - Canvas HTML5 para mostrar frames
   - Decodificación de JPEG
   - Escalado y responsive design

2. **Controles de Input**
   - Eventos de mouse y teclado
   - Mapeo a comandos WebSocket
   - Feedback visual de interacciones

3. **UI de Control de Sesión**
   - Lista de sesiones activas
   - Botones de iniciar/terminar
   - Estado de conexión en tiempo real

### Optimizaciones Futuras
1. **Compresión de Frames**
   - Algoritmos de diferencia entre frames
   - Compresión adaptativa según bandwidth
   - Calidad dinámica según latencia

2. **Batching de Comandos**
   - Agrupación de comandos frecuentes
   - Reducción de overhead de red
   - Sincronización temporal

## Estado Actual

**FASE 5 PASO 2: COMPLETADO AL 100%**

✅ **DTOs extendidos** con ScreenFrame e InputCommand  
✅ **RemoteSessionService** con validaciones de seguridad  
✅ **WebSocketHandler** procesa y reenvía screen frames  
✅ **AdminWebSocketHandler** procesa y reenvía input commands  
✅ **Inicialización corregida** con referencias circulares  
✅ **Compilación exitosa** sin errores  
✅ **Logging completo** para debugging y auditoría  
✅ **Validación de seguridad** en todos los flujos  

El sistema está listo para recibir frames del Cliente Wails y comandos del AdminWeb, enrutándolos correctamente según las sesiones activas. 