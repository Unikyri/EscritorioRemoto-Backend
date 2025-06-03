package dto

import "time"

// FileTransferRequest mensaje enviado al cliente para iniciar transferencia
type FileTransferRequest struct {
	Type            string  `json:"type"` // "file_transfer_request"
	TransferID      string  `json:"transfer_id"`
	SessionID       string  `json:"session_id"`
	FileName        string  `json:"file_name"`
	FileSize        int64   `json:"file_size"` // Tamaño en bytes
	FileSizeMB      float64 `json:"file_size_mb"`
	TotalChunks     int     `json:"total_chunks"` // Total de chunks a enviar
	DestinationPath string  `json:"destination_path"`
	InitiatedBy     string  `json:"initiated_by"` // Para logs del servidor
	Timestamp       int64   `json:"timestamp"`    // Unix timestamp
}

// FileChunk mensaje con un chunk del archivo
type FileChunk struct {
	Type          string `json:"type"` // "file_chunk"
	TransferID    string `json:"transfer_id"`
	SessionID     string `json:"session_id"`
	ChunkIndex    int    `json:"chunk_index"` // Índice del chunk (0-based)
	TotalChunks   int    `json:"total_chunks"`
	ChunkData     string `json:"chunk_data"` // Base64 encoded data
	IsLastChunk   bool   `json:"is_last_chunk"`
	ChunkSize     int    `json:"chunk_size"`
	ChunkChecksum string `json:"chunk_checksum,omitempty"`
	Timestamp     int64  `json:"timestamp"` // Unix timestamp
}

// FileTransferAcknowledgement respuesta del cliente sobre el estado de la transferencia
type FileTransferAcknowledgement struct {
	Type         string `json:"type"` // "file_transfer_ack"
	TransferID   string `json:"transfer_id"`
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`  // "COMPLETED_CLIENT", "FAILED_CLIENT", "READY", "CHUNK_RECEIVED"
	Success      bool   `json:"success"` // Para compatibilidad con cliente
	ErrorMessage string `json:"error_message,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	FileChecksum string `json:"file_checksum,omitempty"`
	ChunkNumber  int    `json:"chunk_number,omitempty"` // Para confirmación de chunks
	Timestamp    int64  `json:"timestamp"`              // Unix timestamp
}

// FileTransferProgress mensaje de progreso de transferencia
type FileTransferProgress struct {
	Type              string    `json:"type"` // "file_transfer_progress"
	TransferID        string    `json:"transfer_id"`
	BytesTransferred  int64     `json:"bytes_transferred"`
	TotalBytes        int64     `json:"total_bytes"`
	PercentComplete   float64   `json:"percent_complete"`
	ChunksTransferred int       `json:"chunks_transferred"`
	TotalChunks       int       `json:"total_chunks"`
	Timestamp         time.Time `json:"timestamp"`
}

// FileTransferStatus mensaje de estado general de transferencia
type FileTransferStatus struct {
	Type         string    `json:"type"` // "file_transfer_status"
	TransferID   string    `json:"transfer_id"`
	Status       string    `json:"status"` // "PENDING", "IN_PROGRESS", "COMPLETED", "FAILED"
	ErrorMessage string    `json:"error_message,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	EndTime      time.Time `json:"end_time,omitempty"`
	Duration     string    `json:"duration,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}
