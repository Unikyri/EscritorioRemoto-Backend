-- Script de migración para arreglar el campo associated_session_id
-- Ejecutar este script en la base de datos existente

USE escritorio_remoto_db;

-- Aumentar el tamaño del campo associated_session_id de VARCHAR(36) a VARCHAR(50)
ALTER TABLE session_videos 
MODIFY COLUMN associated_session_id VARCHAR(50) NOT NULL;

-- Verificar el cambio
DESCRIBE session_videos;

SELECT 'Campo associated_session_id actualizado exitosamente a VARCHAR(50)' as mensaje; 