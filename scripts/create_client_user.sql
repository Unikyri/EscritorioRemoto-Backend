-- Script para crear un usuario cliente de prueba
-- Contraseña: "cliente123" (hasheada con bcrypt)

INSERT INTO users (user_id, username, ip, hashed_password, role, is_active, created_at, updated_at) 
VALUES (
    '550e8400-e29b-41d4-a716-446655440002',
    'cliente1',
    '192.168.1.100',
    '$2a$10$qGqrah.g80sJC40GarRwe.9T.DE.8jQ8AyZ/QROP23ZtyqufQzn2e', -- password: cliente123
    'CLIENT_USER',
    true,
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE
    username = VALUES(username),
    ip = VALUES(ip),
    hashed_password = VALUES(hashed_password),
    role = VALUES(role),
    is_active = VALUES(is_active),
    updated_at = NOW();

-- Verificar que se insertó correctamente
SELECT user_id, username, ip, role, is_active, created_at 
FROM users 
WHERE username = 'cliente1'; 