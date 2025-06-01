
# Proyecto: Administración Remota de Equipos de Cómputo - Backend

## 1. Descripción General

Este repositorio contiene el código fuente y la documentación del componente Backend (Servidor) para el sistema de Administración Remota de Equipos de Cómputo. Es responsable de gestionar la lógica de negocio principal, la comunicación con la base de datos, la autenticación de usuarios, el manejo de conexiones WebSocket con los clientes y la interfaz de administración web, y la orquestación general de las operaciones remotas.
## 2. Tecnologías Utilizadas

* **Go (Golang):** Lenguaje principal para el desarrollo del servidor. Se utiliza para:
    * Construir la API HTTP/REST para la interfaz de administración web y algunas operaciones del cliente.
    * Manejar conexiones WebSocket (WSS) para la comunicación en tiempo real con la Aplicación Cliente y el Frontend de Administración Web (para control remoto, logs, etc.).
    * Implementar toda la lógica de negocio (capas de Aplicación y Dominio).
* **MySQL:** Sistema de gestión de bases de datos relacional para el almacenamiento persistente de datos desplegado en Docker.
* **Redis:** Almacén de datos en memoria utilizado para cachear información frecuentemente accedida y mejorar el rendimiento, como metadatos para informes desplegado en Docker.
* **WebSockets (WSS):** Protocolo para comunicación bidireccional en tiempo real entre el servidor y los clientes (AdminWeb y Aplicación Cliente).

## 3. Requerimientos Específicos del Backend

* **Gestión de Usuarios:**
    * Permitir que un Administrador cree cuentas para nuevos Usuarios Cliente (Correo, password).
    * Debe existir al menos una cuenta de Administrador predefinida o creada por un script inicial.
* **Gestión de PCs Cliente:**
    * Recibir y procesar el registro de PCs de los Usuarios Cliente.
    * Proveer una lista de los PCs Cliente registrados y su estado (conectado/desconectado) a la interfaz de administración.
    * Gestionar el inicio de sesiones de control remoto sobre un PC Cliente conectado.
    * Recibir el stream de imágenes del escritorio del PC Cliente y transmitirlo a la interfaz de administración (AdminWeb).
    * Recibir eventos de input (mouse/teclado) desde la interfaz de administración (AdminWeb) y transmitirlos al PC Cliente.
* **Registro de Acciones y Auditoría (Log):**
    * Registrar eventos básicos: inicio/fin de sesión de usuarios, inicio/fin de sesión de control remoto, transferencias de archivos.
    * Almacenar estos logs en la base de datos MySQL.
    * Proveer una interfaz CLI para mostrar logs en tiempo real.
* **Grabación de Sesión en Video:**
    * Recibir los archivos de video grabados desde la aplicación cliente.
    * Almacenar los videos en un directorio configurable en el servidor.
    * Guardar la ruta o referencia al video en MySQL.
* **Transferencia de Archivos:**
    * Permitir al Administrador subir un archivo al servidor.
    * Enviar un archivo seleccionado desde el servidor a un directorio predefinido en el PC Cliente.
    * Registrar las transferencias de archivos.
* **Almacenamiento y Caché:**
    * Utilizar tablas en MySQL para usuarios, PCs cliente, sesiones de control, logs de acciones, metadatos de videos y archivos transferidos.
    * Utilizar Redis para cachear metadatos de videos o logs que se consulten frecuentemente.
* **Informes:**
    * Proveer los datos necesarios para que la interfaz de administración genere informes básicos por Usuario Cliente.

## 4. Casos de Uso del Servidor

* **CU-A1: Autenticar Administrador**
Permite a un Administrador iniciar sesión en el sistema web proporcionando sus credenciales.
Actor Primario: Administrador
Flujo Principal: 
○ El Administrador navega a la página de inicio de sesión del sistema web. 
○ El Administrador ingresa su correo y contraseña. 
○ El Administrador envía el formulario de inicio de sesión. 
○ El Sistema verifica las credenciales contra la base de datos. 
○ Si las credenciales son válidas, el Sistema crea una sesión para el Administrador y le redirige al panel de control principal. 
● Postcondiciones (Éxito): 
○ El Administrador ha iniciado sesión en el sistema. 
○ Se ha establecido una sesión válida para el Administrador. 
● Flujos Alternativos/Excepciones: 
○ Credenciales Inválidas: El Sistema muestra un mensaje de error. El Administrador permanece en la página de inicio de sesión.

* **CU-A2: Gestionar Cuentas de Usuario Cliente**
Permite a un Administrador crear nuevas cuentas para Usuarios Cliente
Actor Primario: Administrador
Flujo Principal (Crear Usuario Cliente): 
○ El Administrador navega a la sección de gestión de usuarios. 
○ El Administrador selecciona la opción para crear un nuevo Usuario Cliente. 
○ El Administrador ingresa el nombre de usuario y la contraseña para el nuevo Usuario Cliente. 
○ El Administrador envía el formulario. 
○ El Sistema valida los datos (ej: nombre de usuario no duplicado). 
○ El Sistema crea la nueva cuenta de Usuario Cliente en la base de datos. 
○ El Sistema muestra un mensaje de confirmación. 
● Postcondiciones (Éxito al Crear): 
○ Se ha creado una nueva cuenta de Usuario Cliente en el sistema. 
● Flujos Alternativos/Excepciones: 
○ Nombre de Usuario Duplicado: El Sistema muestra un mensaje de error. 
○ Datos Inválidos: El Sistema muestra un mensaje de error indicando el campo incorrecto.

* **CU-A3: Visualizar Lista de PCs Cliente**
Permite al Administrador ver una lista de todos los PCs Cliente registrados y su estado de conexión.
Actor Primario: Administrador

Flujo Principal: 
○ El Administrador navega a la sección de visualización de PCs Cliente en el panel de control. 
○ El Sistema recupera la lista de PCs Cliente registrados desde la base de datos y sus estados actuales de conexión (obtenidos a través de las conexiones WebSocket activas o un último estado conocido). 
○ El Sistema muestra la lista de PCs (ej: Nombre del PC/Identificador, Estado Online/Offline). 
● Postcondiciones (Éxito): 
○ El Administrador visualiza la lista de PCs Cliente y su estado. 

* **CU-A4: Administrar Sesión de Control Remoto**
 Permite al Administrador iniciar, visualizar, controlar y finalizar una sesión de control remoto sobre un PC Cliente.
Flujo Principal: 
○ El Administrador selecciona un PC Cliente online de la lista (CU-A3) y elige la opción "Iniciar Control Remoto". 
○ El Sistema Servidor envía una solicitud de control remoto al Sistema Cliente del PC seleccionado. 
○ (Flujo en el Sistema Cliente - Ver CU-C3 para detalles) El Usuario Cliente acepta la solicitud. 
○ El Sistema Cliente notifica al Sistema Servidor la aceptación. 
○ El Sistema Servidor establece el canal de streaming de video desde el Sistema Cliente hacia la interfaz web del Administrador. 
○ El Sistema Servidor establece el canal para enviar eventos de mouse/teclado desde la interfaz web del Administrador hacia el Sistema Cliente. 
○ El Administrador visualiza el escritorio del PC Cliente y puede enviar eventos de mouse/teclado. 
○ El Sistema Cliente comienza la grabación de la sesión en video (Ver CU-C4). 
○ El Administrador realiza las acciones necesarias en el PC Cliente. 
○ El Administrador selecciona la opción "Finalizar Sesión de Control Remoto". 
○ El Sistema Servidor envía una señal de finalización al Sistema Cliente. 
○ Se cierran los canales de streaming y control. 
○ El Sistema Cliente detiene la grabación y comienza el envío del video al servidor (Ver CU-C4). 
○ El Sistema Servidor registra el fin de la sesión. 
● Postcondiciones (Éxito): 
○ La sesión de control remoto se ha completado. 
○ Se ha registrado el inicio y fin de la sesión. 
○ El video de la sesión está en proceso de ser enviado o ha sido 
enviado al servidor. 
● Flujos Alternativos/Excepciones: 
○ PC Cliente se Desconecta: La sesión se interrumpe. El Sistema lo notifica al Administrador. 
○ Usuario Cliente Rechaza la Solicitud: El Sistema notifica al Administrador. La sesión no se inicia. 
○ Error en el Streaming/Control: La sesión podría interrumpirse o funcionar con degradación. 

* **CU-A5: Transferir Archivo a PC Cliente**
Permite al Administrador transferir un archivo desde el servidor a un PC Cliente durante una sesión de control remoto.
Precondiciones: 
○ El Administrador ha iniciado sesión (CU-A1). 
○ Una sesión de control remoto está activa con un PC Cliente 
(CU-A4). 
● Flujo Principal: 
○ Durante una sesión de control remoto activa, el Administrador selecciona la opción "Transferir Archivo". 
○ La interfaz web permite al Administrador seleccionar un archivo local (para subirlo temporalmente al servidor) o un archivo ya existente en una ubicación designada del servidor. 
○ El Administrador confirma el archivo a enviar. 
○ El Sistema Servidor inicia la transferencia del archivo al Sistema Cliente a través del canal de comunicación establecido (WebSocket o un canal dedicado para archivos). 
○ El Sistema Cliente recibe el archivo y lo guarda en un directorio predefinido. 
○ El Sistema Cliente notifica al Sistema Servidor la recepción exitosa (o fallo). 
○ El Sistema Servidor notifica al Administrador el resultado de la transferencia. 
○ El Sistema Servidor registra la transferencia de archivo. 
● Postcondiciones (Éxito): 
○ El archivo ha sido transferido al directorio predefinido en el PC Cliente. 
○ La transferencia ha sido registrada. 
● Flujos Alternativos/Excepciones: 
○ Error de Transferencia: El archivo no se transfiere. El Sistema notifica al Administrador. 
○ PC Cliente sin Espacio (Fuera de MVP la verificación previa): La transferencia falla en el cliente. 
○ Conexión Interrumpida: La transferencia falla.

* **CU-A6: Consultar Logs de Auditoría** (tanto vía web como CLI)
Permite al Administrador ver los logs de acciones importantes del sistema.
Flujo Principal (Interfaz Web): 
○ El Administrador navega a la sección de visualización de logs. 
○ El Administrador puede aplicar filtros básicos (ej: por fecha, por tipo de evento - MVP básico). 
○ El Sistema recupera los logs de la base de datos MySQL (Con ayuda de Redis para consultas frecuentes). 
○ El Sistema muestra los logs al Administrador. 
● Flujo Principal (CLI del Servidor - para HU-A12): 
○ El operador del servidor ejecuta el comando para ver logs en la CLI. 
○ La aplicación servidor muestra los logs relevantes en tiempo real 
o históricos según el comando. 
● Postcondiciones (Éxito): 
○ El Administrador visualiza los logs solicitados.

* **CU-A7: Generar y Visualizar Informes por Usuario Cliente**
Permite al Administrador ver informes de actividad para un Usuario Cliente específico, incluyendo sesiones de control, vídeos y transferencias.
El Administrador navega a la sección de informes. 
○ El Administrador selecciona un Usuario Cliente para el cual 
generar el informe. 
○ El Sistema recupera de MySQL (Redis para caché) la información de: 
     ■ Sesiones de control remoto (fecha, duración, admin que controló). 
     ■ Metadatos de videos asociados a esas sesiones (nombre, enlace para descarga). 
     ■ Transferencias de archivos realizadas a los PCs de ese usuario. 
○ El Sistema presenta el informe de forma estructurada al Administrador. 
○ Si el Administrador selecciona un video, el Sistema le ofrece la opción de descargarlo. 
● Postcondiciones (Éxito): 
○ El Administrador visualiza el informe del Usuario Cliente. 
○ El Administrador puede descargar los videos de sesión. 




## 5. Modelos de Datos (Capas del Servidor)

El backend sigue una arquitectura por capas. Los modelos de datos relevantes se encuentran en:

* **Capa de Dominio:** Contiene las entidades, agregados, objetos de valor y servicios de dominio.
    ``` classDiagram 
    direction LR 
 
    class User { 
        +UserID 
        +Username 
        +Role
        +IP
    } 
 
    class Administrator { 
        <<Subtype>> 
        %% Inherits from User 
    } 
 
    class UserClient { 
        <<Subtype>> 
        %% Inherits from User 
    } 
 
    class ClientPC { 
        +PC_ID 
        +IP 
        +ConnectionStatus 
    } 
 
    class RemoteSession { 
        +SessionID 
        +StartTime 
        +EndTime 
    } 
 
    class SessionVideo { 
        +VideoID 
        +FilePath 
        +Duration 
    } 
 
    class FileTransfer { 
        +TransferID 
        +FileName 
        +Status 
    } 
 
    class ActionLog { 
        +LogID 
        +Timestamp 
        +ActionType 
        +Description 
        +PerformedByUserID 
        +SubjectEntityID (optional) 
    } 
 
    User <|-- Administrator 
    User <|-- UserClient 
 
    UserClient "1" -- "1..*" ClientPC : Registers/Operates 
    Administrator "1" -- "0..*" RemoteSession : Initiates 
    RemoteSession "1" -- "1" ClientPC : Targets 
    RemoteSession "1" -- "1" SessionVideo : Produces 
    RemoteSession "1" -- "0..*" FileTransfer : Involves 
 
    %% ActionLog relationships can be broad. 
    %% A User performs an action that is logged. 
    %% The log entry itself might reference other entities. 
User "1" -- "0..*" ActionLog : IsActorFor 
%% Optional conceptual links showing what an ActionLog might be 
about 
ActionLog "0..*" -- "0..1" RemoteSession : DescribesActionOn 
ActionLog "0..*" -- "0..1" FileTransfer : DescribesActionOn 
ActionLog "0..*" -- "0..1" ClientPC : DescribesActionOn 
ActionLog "0..*" -- "0..1" User : DescribesActionOnUser
```

* **Capa de Aplicación:** Contiene los servicios de aplicación (casos de uso) y las interfaces de los repositorios.
    * Ver Diagrama de Clases - Capa de Aplicación.
    ```plantuml
    @startuml Application Layer Class Diagram
    !theme materia-outline
    '--- Contenido del diagrama de clases de la capa de Aplicación
    ' Ejemplo:
    interface IUserRepository {
      +FindById(id: string): User
      +Save(user: User): void
    }
    class UserService {
      -userRepository: IUserRepository
      +RegisterClientUser(username: string, password: string): UserDTO
    }
    ' ... (resto de las interfaces y clases de aplicación)
    @enduml
    ```
* **Capa de Presentación:** Manejadores HTTP, manejadores WebSocket, DTOs.
    * Ver Diagrama de Clases - Capa de Presentación.
    ```plantuml
    @startuml Presentation Layer Class Diagram
    !theme materia-outline
    '--- Contenido del diagrama de clases de la capa de Presentación (pg 35-41) ---
    ' Ejemplo:
    class AuthHandler {
      -userService: UserService
      +Login(req: AuthRequestDTO): AuthResultDTO
    }
    class AuthRequestDTO {
      +Username: string
      +Password: string
    }
    ' ... (resto de las clases de presentación y DTOs)
    @enduml
    ```
* **Capa de Infraestructura:** Implementaciones de repositorios, servicios de caché, almacenamiento de archivos.
    * Ver Diagrama de Clases - Capa de Infraestructura.
    ```plantuml
    @startuml Infrastructure Layer Class Diagram
    !theme materia-outline
    '--- Contenido del diagrama de clases de la capa de Infraestructura (pg 41-46) ---
    ' Ejemplo:
    class MySQLUserRepository implements IUserRepository {
      -dbConnection: MySQLConnection
      +FindById(id: string): User
    }
    class RedisCacheService implements ICacheService {
      -redisClient: RedisClient
    }
    ' ... (resto de las clases de infraestructura)
    @enduml
    ```
* **Utilidades Compartidas:** Módulos para configuración, manejo de errores, patrones comunes.
    * Ver Diagrama de Clases - Capa de Utilidades.
    ```plantuml
    @startuml Shared Utilities Layer Class Diagram
    !theme materia-outline
    '--- Contenido del diagrama de clases de la capa de Utilidades (pg 46-50) ---
    class ConfigProvider {
      +GetString(key: string): string
    }
    ' ... (resto de las clases de utilidades)
    @enduml
    ```



## 6. Diagrama Relacional (MySQL)

A continuación, el script SQL conceptual para la creación de las tablas en MySQL, basado en el diagrama relacional. [cite: 319, 320]

```sql
-- users Table
CREATE TABLE users (
    user_id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    ip VARCHAR(255) NOT NULL,
    hashed_password VARCHAR(255) NOT NULL,
    role ENUM('ADMINISTRATOR', 'CLIENT_USER') NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
); 

-- client_pcs Table
CREATE TABLE client_pcs (
    pc_id VARCHAR(36) PRIMARY KEY,
    ip VARCHAR(255) NOT NULL,
    connection_status ENUM('ONLINE', 'OFFLINE', 'CONNECTING') DEFAULT 'OFFLINE',
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    owner_user_id VARCHAR(36) NOT NULL,
    last_seen_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_user_id) REFERENCES users(user_id) ON DELETE CASCADE
); 

-- remote_sessions Table
CREATE TABLE remote_sessions (
    session_id VARCHAR(36) PRIMARY KEY,
    admin_user_id VARCHAR(36) NOT NULL,
    client_pc_id VARCHAR(36) NOT NULL,
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    status ENUM('PENDING_APPROVAL', 'ACTIVE', 'ENDED_SUCCESSFULLY', 'ENDED_BY_ADMIN', 'ENDED_BY_CLIENT', 'FAILED') NOT NULL,
    session_video_id VARCHAR(36) NULL, -- FK a session_videos
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (admin_user_id) REFERENCES users(user_id),
    FOREIGN KEY (client_pc_id) REFERENCES client_pcs(pc_id)
    -- FOREIGN KEY (session_video_id) REFERENCES session_videos(video_id) ON DELETE SET NULL -- Añadido después de crear session_videos
); 

-- session_videos Table
CREATE TABLE session_videos (
    video_id VARCHAR(36) PRIMARY KEY,
    file_path VARCHAR(1024) NOT NULL,
    duration_seconds INT,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    associated_session_id VARCHAR(36) NOT NULL,
    file_size_mb FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (associated_session_id) REFERENCES remote_sessions(session_id) ON DELETE CASCADE
); 

-- Alter remote_sessions to add the foreign key for session_video_id
ALTER TABLE remote_sessions
ADD CONSTRAINT fk_session_video
FOREIGN KEY (session_video_id) REFERENCES session_videos(video_id) ON DELETE SET NULL;


-- file_transfers Table
CREATE TABLE file_transfers (
    transfer_id VARCHAR(36) PRIMARY KEY,
    file_name VARCHAR(255) NOT NULL,
    source_path_server VARCHAR(1024),
    destination_path_client VARCHAR(1024),
    transfer_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED') NOT NULL,
    associated_session_id VARCHAR(36) NOT NULL,
    initiating_user_id VARCHAR(36) NOT NULL, -- Admin que inició
    target_pc_id VARCHAR(36) NOT NULL,
    file_size_mb FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (associated_session_id) REFERENCES remote_sessions(session_id),
    FOREIGN KEY (initiating_user_id) REFERENCES users(user_id),
    FOREIGN KEY (target_pc_id) REFERENCES client_pcs(pc_id)
); 

-- action_logs Table
CREATE TABLE action_logs (
    log_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    action_type ENUM('USER_LOGIN', 'USER_LOGOUT', 'USER_CREATED', 'PC_REGISTERED', 'PC_STATUS_CHANGED', 'REMOTE_SESSION_STARTED', 'REMOTE_SESSION_ENDED', 'FILE_TRANSFER_INITIATED', 'FILE_TRANSFER_COMPLETED', 'FILE_TRANSFER_FAILED', 'VIDEO_RECORDING_STARTED', 'VIDEO_RECORDING_ENDED', 'VIDEO_UPLOADED') NOT NULL,
    description TEXT,
    performed_by_user_id VARCHAR(36) NOT NULL,
    subject_entity_id VARCHAR(255) NULL,
    subject_entity_type VARCHAR(100) NULL,
    details JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (performed_by_user_id) REFERENCES users(user_id)
);

El proyecto deberá cumplir con principios SOLID y se deben implementar patrones de diseño como DTO, DAO, REPOSITORY, FACTORY, OBSERVER (IMPORTANTE IMPLEMENTAR ESTE), CONTROLLER
/ (raíz del proyecto backend)
|-- cmd/
|   |-- server/                 # Punto de entrada principal para la aplicación servidor
|       |-- main.go
|-- internal/                   # Código privado del proyecto, no importable por otros
|   |-- shared/                 # util.config, util.errors, util.patterns [cite: 234]
|   |   |-- config/
|   |   |-- errors/
|   |   |-- patterns/
|   |-- domain/                 # Entidades, VOs, servicios de dominio, eventos, fábricas [cite: 234]
|   |   |-- user/
|   |   |-- clientpc/
|   |   |-- remotesession/
|   |   |-- ... (otras entidades)
|   |-- application/            # Servicios de aplicación (casos de uso), interfaces de repo [cite: 234]
|   |   |-- userservice/
|   |   |-- pcservice/
|   |   |-- ... (otros servicios de aplicación)
|   |   |-- interfaces/         # Interfaces (repositorios, caché, archivos, video) [cite: 22]
|   |-- presentation/           # Handlers (HTTP, WebSocket), DTOs, Middleware [cite: 22]
|   |   |-- handlers/
|   |   |-- dto/
|   |   |-- middleware/
|   |-- infrastructure/         # Implementaciones (MySQL, Redis, Filesystem, WebSocket infra, DI) [cite: 22]
|       |-- persistence/
|       |   |-- mysql/
|       |-- cache/
|       |   |-- redis/
|       |-- storage/
|       |   |-- filesystem/
|       |-- media/
|       |   |-- videoprocessing/
|       |-- comms/
|       |   |-- websocket/
|       |-- platform/
|           |-- di/
|-- pkg/                        # Código público (si se planea que otras apps lo importen, sino usar `internal`)
|-- api/                        # Definiciones de API (ej. OpenAPI/Swagger si se usa)
|-- configs/                    # Archivos de configuración (ej. config.yaml.example)
|-- scripts/                    # Scripts útiles (ej. creación de admin inicial, migraciones)
|-- test/                       # Tests (pueden estar también dentro de cada paquete)
|-- go.mod
|-- go.sum
|-- README.md
