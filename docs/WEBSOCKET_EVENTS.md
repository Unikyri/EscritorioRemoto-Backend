# Eventos WebSocket para AdminWeb - Observer Pattern

## 🎯 **Objetivo**
Eliminar el polling constante del AdminWeb y reemplazarlo con **notificaciones en tiempo real** usando el patrón Observer.

## 📡 **Eventos Disponibles**

### 1. **`pc_registered`** - Nuevo PC Registrado
```json
{
  "type": "pc_registered",
  "data": {
    "pcId": "f5899f37-f80b-46c0-8134-421467f243eb",
    "identifier": "Daikyri-PC-windows",
    "ownerUserId": "550e8400-e29b-41d4-a716-446655440002",
    "ip": "127.0.0.1",
    "status": "ONLINE",
    "timestamp": 1672531200,
    "event": "registration"
  }
}
```

### 2. **`pc_connected`** - PC Se Conectó
```json
{
  "type": "pc_connected", 
  "data": {
    "pcId": "f5899f37-f80b-46c0-8134-421467f243eb",
    "identifier": "Daikyri-PC-windows",
    "ownerUserId": "550e8400-e29b-41d4-a716-446655440002",
    "ip": "127.0.0.1",
    "status": "ONLINE",
    "timestamp": 1672531200,
    "event": "connection"
  }
}
```

### 3. **`pc_disconnected`** - PC Se Desconectó
```json
{
  "type": "pc_disconnected",
  "data": {
    "pcId": "f5899f37-f80b-46c0-8134-421467f243eb",
    "identifier": "Daikyri-PC-windows", 
    "ownerUserId": "550e8400-e29b-41d4-a716-446655440002",
    "status": "OFFLINE",
    "timestamp": 1672531200,
    "event": "disconnection"
  }
}
```

### 4. **`pc_status_changed`** - Cambio de Estado
```json
{
  "type": "pc_status_changed",
  "data": {
    "pcId": "f5899f37-f80b-46c0-8134-421467f243eb", 
    "identifier": "Daikyri-PC-windows",
    "oldStatus": "OFFLINE",
    "newStatus": "ONLINE", 
    "timestamp": 1672531200,
    "event": "status_change"
  }
}
```

### 5. **`pc_list_update`** - Lista Debe Actualizarse
```json
{
  "type": "pc_list_update",
  "data": {
    "timestamp": 1672531200,
    "event": "list_refresh", 
    "message": "PC list has been updated, please refresh"
  }
}
```

## 🔧 **Implementación en AdminWeb**

### WebSocket Service (ya implementado)
```typescript
// Suscribirse a eventos específicos
websocketService.subscribe('pc_registered', (event) => {
  // Agregar PC a la lista local
  addPCToList(event.data);
});

websocketService.subscribe('pc_connected', (event) => {
  // Actualizar estado del PC a ONLINE
  updatePCStatus(event.data.pcId, 'ONLINE');
});

websocketService.subscribe('pc_disconnected', (event) => {
  // Actualizar estado del PC a OFFLINE  
  updatePCStatus(event.data.pcId, 'OFFLINE');
});

websocketService.subscribe('pc_status_changed', (event) => {
  // Actualizar estado específico
  updatePCStatus(event.data.pcId, event.data.newStatus);
});

websocketService.subscribe('pc_list_update', (event) => {
  // Refrescar lista completa desde API (solo si es necesario)
  refreshPCList();
});
```

### Funciones Sugeridas para AdminWeb
```typescript
// Agregar PC a la lista local sin API call
function addPCToList(pcData: any) {
  // Agregar al store/state local de PCs
  pcStore.update(pcs => [...pcs, pcData]);
}

// Actualizar estado de PC sin API call  
function updatePCStatus(pcId: string, newStatus: string) {
  pcStore.update(pcs => 
    pcs.map(pc => 
      pc.pcId === pcId 
        ? { ...pc, connectionStatus: newStatus }
        : pc
    )
  );
}

// Solo refrescar desde API cuando sea absolutamente necesario
function refreshPCList() {
  // Llamar API solo como fallback
  pcService.getAllPCs().then(pcs => {
    pcStore.set(pcs);
  });
}
```

## ✅ **Beneficios del Observer Pattern**

1. **Eliminación de Polling**: No más peticiones constantes al servidor
2. **Actualizaciones Instantáneas**: UI se actualiza inmediatamente cuando algo cambia
3. **Mejor Performance**: Menos carga en servidor y cliente
4. **No Más Parpadeo**: UI estable y sin refreshes constantes
5. **Eliminación de Botones de Recarga**: No necesarios con actualizaciones automáticas

## 🚀 **Estados de Conexión**

| Estado | Descripción |
|--------|-------------|
| `ONLINE` | PC conectado y activo |
| `OFFLINE` | PC desconectado |
| `CONNECTING` | PC en proceso de conexión |

## 📝 **Notas de Implementación**

- Todos los eventos incluyen `timestamp` para ordenamiento
- Campo `event` ayuda a clasificar el tipo de evento
- `pcId` siempre incluido para identificación única
- `identifier` incluido cuando está disponible para UI
- `ownerUserId` para filtrado por usuario si es necesario

## 🔄 **Flujo Típico**

1. **PC se registra por primera vez**:
   - `pc_registered` → Agregar a lista
   - `pc_connected` → Marcar como ONLINE  
   - `pc_status_changed` → OFFLINE → ONLINE
   - `pc_list_update` → Confirmar actualización

2. **PC se reconecta**:
   - `pc_connected` → Marcar como ONLINE
   - `pc_status_changed` → OFFLINE → ONLINE
   - `pc_list_update` → Confirmar actualización

3. **PC se desconecta**:
   - `pc_disconnected` → Marcar como OFFLINE
   - `pc_status_changed` → ONLINE → OFFLINE  
   - `pc_list_update` → Confirmar actualización 