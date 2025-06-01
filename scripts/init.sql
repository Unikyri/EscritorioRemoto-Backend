-- Script de inicialización de la base de datos
-- Escritorio Remoto - Schema MySQL

USE escritorio_remoto_db;

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
    session_video_id VARCHAR(36) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (admin_user_id) REFERENCES users(user_id),
    FOREIGN KEY (client_pc_id) REFERENCES client_pcs(pc_id)
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

-- Add FK constraint for session_video_id
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
    initiating_user_id VARCHAR(36) NOT NULL,
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

-- Insertar usuario administrador inicial
-- UUID: admin-000-000-000-000000000001
-- Username: admin
-- Password: Admin123! (hashed con bcrypt)
INSERT INTO users (
    user_id, 
    username, 
    ip, 
    hashed_password, 
    role, 
    is_active,
    created_at,
    updated_at
) VALUES (
    'admin-000-000-000-000000000001',
    'admin',
    '127.0.0.1',
    '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', -- bcrypt hash de "password"
    'ADMINISTRATOR',
    TRUE,
    NOW(),
    NOW()
);

-- Log de creación del usuario administrador
INSERT INTO action_logs (
    action_type,
    description,
    performed_by_user_id,
    subject_entity_id,
    subject_entity_type,
    details
) VALUES (
    'USER_CREATED',
    'Usuario administrador inicial creado automáticamente durante la inicialización del sistema',
    'admin-000-000-000-000000000001',
    'admin-000-000-000-000000000001',
    'USER',
    JSON_OBJECT('initialized_by', 'system', 'role', 'ADMINISTRATOR')
); 