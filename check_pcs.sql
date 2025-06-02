-- Verificar PCs registrados
USE escritorio_remoto_db;

-- Mostrar todos los PCs registrados
SELECT 
    pc_id,
    identifier,
    ip,
    connection_status,
    registered_at,
    owner_user_id,
    last_seen_at
FROM client_pcs 
ORDER BY registered_at DESC;

-- Mostrar usuarios
SELECT user_id, username, role FROM users;

-- Contar PCs por estado
SELECT 
    connection_status, 
    COUNT(*) as count 
FROM client_pcs 
GROUP BY connection_status; 