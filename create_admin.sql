-- Script para crear/actualizar usuario administrador
USE escritorio_remoto_db;

-- Insertar o actualizar usuario admin
INSERT INTO users (user_id, username, ip, hashed_password, role, is_active) 
VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    'admin', 
    '127.0.0.1', 
    '$2a$10$1cyKAIrr0iraJkarFYlHqO2VOoFy72hmaKtMGuXG/1g8aNqfqyqlq', 
    'ADMINISTRATOR', 
    1
) 
ON DUPLICATE KEY UPDATE 
    hashed_password = VALUES(hashed_password), 
    role = VALUES(role),
    is_active = VALUES(is_active);

-- Insertar usuario cliente de prueba
INSERT INTO users (user_id, username, ip, hashed_password, role, is_active) 
VALUES (
    '550e8400-e29b-41d4-a716-446655440001',
    'cliente1', 
    '192.168.1.100', 
    '$2a$10$ovKlOM3vFPRpBTmBX19v3.wOjecWj.wJmGNWJoUs3s9/kkul.XSg2', 
    'CLIENT_USER', 
    1
) 
ON DUPLICATE KEY UPDATE 
    hashed_password = VALUES(hashed_password), 
    role = VALUES(role),
    is_active = VALUES(is_active);

SELECT 'Usuarios creados/actualizados correctamente' as mensaje;
SELECT username, role, is_active FROM users; 