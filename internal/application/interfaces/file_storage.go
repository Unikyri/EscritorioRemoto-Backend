package interfaces

import "context"

// IFileStorage define la interfaz para el almacenamiento de archivos
type IFileStorage interface {
	// SaveFile guarda un archivo en el almacenamiento y retorna la ruta final
	SaveFile(ctx context.Context, destinationPath string, content []byte) (string, error)

	// ReadFile lee un archivo del almacenamiento
	ReadFile(ctx context.Context, filePath string) ([]byte, error)

	// DeleteFile elimina un archivo del almacenamiento
	DeleteFile(ctx context.Context, filePath string) error

	// FileExists verifica si un archivo existe
	FileExists(ctx context.Context, filePath string) bool

	// GetFileSize obtiene el tamaño de un archivo en bytes
	GetFileSize(ctx context.Context, filePath string) (int64, error)

	// CreateDirectory crea un directorio si no existe
	CreateDirectory(ctx context.Context, dirPath string) error

	// GetFilePath construye la ruta completa para un archivo
	GetFilePath(relativePath string) string
}
