-- Script para agregar el campo identifier faltante en la tabla client_pcs
USE escritorio_remoto_db;

-- Agregar la columna identifier después de pc_id
ALTER TABLE client_pcs 
ADD COLUMN identifier VARCHAR(255) NOT NULL DEFAULT 'TEMP_IDENTIFIER' AFTER pc_id;

-- Crear un índice para mejorar las búsquedas por identifier + owner
CREATE INDEX idx_identifier_owner ON client_pcs(identifier, owner_user_id);

-- Verificar la estructura actualizada
DESCRIBE client_pcs;

SELECT 'Tabla client_pcs actualizada correctamente con el campo identifier' as mensaje; 