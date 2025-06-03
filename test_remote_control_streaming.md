# Pruebas de Enrutamiento de Mensajes - Fase 5 Paso 2

## Configuración de Pruebas

### 1. Iniciar el Servidor Backend
```bash
cd EscritorioRemoto-Backend
go run cmd/server/main.go
```

### 2. Herramientas de Prueba Recomendadas
- **wscat**: `npm install -g wscat`
- **websocat**: `cargo install websocat` (alternativa)
- **Postman WebSocket** (interfaz gráfica)
- **Browser DevTools** (para pruebas desde web)

## Casos de Prueba

### Prueba 1: Autenticación y Registro de Cliente

**1.1 Conectar Cliente WebSocket**
```bash
wscat -c ws://localhost:8080/ws/client
```

**1.2 Autenticar Cliente**
```json
{
    "type": "CLIENT_AUTH_REQUEST",
    "data": {
        "username": "testuser",
        "password": "password123"
    }
}
```

**Respuesta Esperada:**
```json
{
    "type": "CLIENT_AUTH_RESPONSE",
    "data": {
        "success": true,
        "token": "jwt-token-here",
        "userId": "user-id-here"
    }
}
```

**1.3 Registrar PC Cliente**
```json
{
    "type": "PC_REGISTRATION_REQUEST",
    "data": {
        "pcIdentifier": "TEST-PC-001",
        "ip": "127.0.0.1"
    }
}
```

**Respuesta Esperada:**
```json
{
    "type": "PC_REGISTRATION_RESPONSE",
    "data": {
        "success": true,
        "pcId": "generated-pc-id"
    }
}
```

### Prueba 2: Autenticación de Administrador

**2.1 Obtener Token de Admin**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

**2.2 Conectar Admin WebSocket**
```bash
wscat -c "ws://localhost:8080/ws/admin?token=YOUR_JWT_TOKEN"
```

**Respuesta Esperada:**
```json
{
    "type": "admin_connected",
    "data": {
        "message": "Connected to admin notifications",
        "adminId": "connection-id"
    }
}
```

### Prueba 3: Iniciar Sesión de Control Remoto

**3.1 Crear Sesión (via HTTP)**
```bash
curl -X POST http://localhost:8080/api/admin/sessions/initiate \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientPCID": "generated-pc-id"
  }'
```

**Respuesta Esperada:**
```json
{
    "success": true,
    "session": {
        "sessionId": "session-uuid",
        "adminUserId": "admin-user-id",
        "clientPCId": "generated-pc-id",
        "status": "PENDING"
    }
}
```

**3.2 Cliente Acepta Sesión**
```json
{
    "type": "session_accepted",
    "data": {
        "session_id": "session-uuid"
    }
}
```

**Respuesta del Servidor al Cliente:**
```json
{
    "type": "session_started",
    "data": {
        "session_id": "session-uuid",
        "status": "ACTIVE",
        "message": "Remote control session started successfully",
        "timestamp": 1640995200
    }
}
```

### Prueba 4: Envío de Screen Frames

**4.1 Cliente Envía Frame de Pantalla**
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
        "frame_data": "base64-encoded-image-data-here",
        "sequence_num": 1
    }
}
```

**4.2 Verificar en Admin WebSocket**
El administrador conectado debe recibir:
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
        "frame_data": "base64-encoded-image-data-here",
        "sequence_num": 1
    }
}
```

### Prueba 5: Envío de Input Commands

**5.1 Admin Envía Comando de Mouse**
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

**5.2 Verificar en Cliente WebSocket**
El cliente debe recibir:
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

**5.3 Admin Envía Comando de Teclado**
```json
{
    "type": "input_command",
    "data": {
        "session_id": "session-uuid",
        "timestamp": 1640995200000,
        "event_type": "keyboard",
        "action": "keydown",
        "payload": {
            "key": "Enter",
            "code": "Enter",
            "modifiers": []
        }
    }
}
```

### Prueba 6: Validación de Seguridad

**6.1 Frame Sin Sesión Activa**
- Enviar frame con `session_id` inválido
- **Resultado Esperado**: Frame rechazado, log de error

**6.2 Comando Sin Permisos**
- Admin envía comando para sesión de otro admin
- **Resultado Esperado**: Comando rechazado, log de error

**6.3 Cliente No Autenticado**
- Enviar frame sin autenticación previa
- **Resultado Esperado**: Frame rechazado, log de error

### Prueba 7: Manejo de Errores

**7.1 JSON Malformado**
```json
{
    "type": "screen_frame",
    "data": {
        "session_id": "session-uuid"
        // JSON incompleto sin coma
        "timestamp": 1640995200000
    }
}
```
- **Resultado Esperado**: Error de parsing, conexión mantenida

**7.2 Tipo de Mensaje Desconocido**
```json
{
    "type": "unknown_message_type",
    "data": {}
}
```
- **Resultado Esperado**: Log "Unknown message type", mensaje ignorado

## Scripts de Prueba Automatizada

### Script 1: Cliente Simulado (Node.js)
```javascript
const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8080/ws/client');

ws.on('open', function open() {
    console.log('Connected as client');
    
    // Autenticar
    ws.send(JSON.stringify({
        type: 'CLIENT_AUTH_REQUEST',
        data: {
            username: 'testuser',
            password: 'password123'
        }
    }));
});

ws.on('message', function message(data) {
    const msg = JSON.parse(data);
    console.log('Client received:', msg.type);
    
    if (msg.type === 'CLIENT_AUTH_RESPONSE' && msg.data.success) {
        // Registrar PC
        ws.send(JSON.stringify({
            type: 'PC_REGISTRATION_REQUEST',
            data: {
                pcIdentifier: 'TEST-PC-SCRIPT',
                ip: '127.0.0.1'
            }
        }));
    }
    
    if (msg.type === 'remote_control_request') {
        // Aceptar automáticamente
        ws.send(JSON.stringify({
            type: 'session_accepted',
            data: {
                session_id: msg.data.session_id
            }
        }));
    }
    
    if (msg.type === 'session_started') {
        // Empezar a enviar frames
        sendFrames(msg.data.session_id);
    }
    
    if (msg.type === 'input_command') {
        console.log('Received input command:', msg.data.action);
    }
});

function sendFrames(sessionId) {
    setInterval(() => {
        ws.send(JSON.stringify({
            type: 'screen_frame',
            data: {
                session_id: sessionId,
                timestamp: Date.now(),
                width: 1920,
                height: 1080,
                format: 'jpeg',
                quality: 75,
                frame_data: Buffer.from('fake-image-data').toString('base64'),
                sequence_num: Math.floor(Math.random() * 1000)
            }
        }));
    }, 1000/15); // 15 FPS
}
```

### Script 2: Admin Simulado (Node.js)
```javascript
const WebSocket = require('ws');

// Reemplazar con token real obtenido de /api/auth/login
const token = 'YOUR_JWT_TOKEN';
const ws = new WebSocket(`ws://localhost:8080/ws/admin?token=${token}`);

ws.on('open', function open() {
    console.log('Connected as admin');
});

ws.on('message', function message(data) {
    const msg = JSON.parse(data);
    console.log('Admin received:', msg.type);
    
    if (msg.type === 'screen_frame') {
        console.log(`Received frame ${msg.data.sequence_num} (${msg.data.width}x${msg.data.height})`);
        
        // Enviar comando de mouse aleatorio
        if (Math.random() > 0.8) {
            sendRandomMouseCommand(msg.data.session_id);
        }
    }
});

function sendRandomMouseCommand(sessionId) {
    ws.send(JSON.stringify({
        type: 'input_command',
        data: {
            session_id: sessionId,
            timestamp: Date.now(),
            event_type: 'mouse',
            action: 'move',
            payload: {
                x: Math.floor(Math.random() * 1920),
                y: Math.floor(Math.random() * 1080)
            }
        }
    }));
}
```

## Logs Esperados

### Cliente Exitoso
```
📹 SCREEN FRAME: Received frame 123 from PC pc-id (session: session-uuid, size: 1920x1080)
✅ SCREEN FRAME: Frame 123 forwarded to admin admin-user-id
```

### Admin Exitoso
```
🎮 INPUT COMMAND: Received from admin admin-username: type=mouse, action=click, session=session-uuid
✅ INPUT COMMAND: Command forwarded to client pc-id
```

### Errores de Validación
```
❌ SCREEN FRAME: Invalid streaming permission: session is not active
❌ INPUT COMMAND: Invalid permission for admin admin-username: admin user ID mismatch
```

## Verificación de Resultados

1. **Conectividad**: Ambos WebSockets conectan sin errores
2. **Autenticación**: Tokens válidos y respuestas exitosas
3. **Enrutamiento**: Mensajes llegan al destinatario correcto
4. **Validación**: Mensajes inválidos son rechazados apropiadamente
5. **Performance**: Sin lag significativo en el enrutamiento
6. **Logs**: Información detallada para debugging

## Métricas de Éxito

- ✅ **Latencia < 50ms** para enrutamiento de mensajes
- ✅ **0% pérdida** de frames durante pruebas de 5 minutos
- ✅ **100% rechazo** de mensajes sin permisos
- ✅ **Reconexión automática** tras desconexión temporal
- ✅ **Limpieza correcta** de recursos al cerrar conexiones 