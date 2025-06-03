package interfaces

import (
	"context"

	"github.com/unikyri/escritorio-remoto-backend/internal/domain/sessionvideo"
)

// ISessionVideoRepository define la interfaz para la persistencia de videos de sesión
type ISessionVideoRepository interface {
	// Save guarda un nuevo video de sesión
	Save(ctx context.Context, video *sessionvideo.SessionVideo) error

	// FindByID busca un video por su ID
	FindByID(ctx context.Context, videoID string) (*sessionvideo.SessionVideo, error)

	// FindBySessionID busca videos asociados a una sesión específica
	FindBySessionID(ctx context.Context, sessionID string) ([]*sessionvideo.SessionVideo, error)

	// Update actualiza un video existente
	Update(ctx context.Context, video *sessionvideo.SessionVideo) error

	// Delete elimina un video por su ID
	Delete(ctx context.Context, videoID string) error

	// FindAll obtiene todos los videos con paginación
	FindAll(ctx context.Context, limit, offset int) ([]*sessionvideo.SessionVideo, error)

	// Count obtiene el total de videos
	Count(ctx context.Context) (int64, error)

	// FindByDateRange busca videos en un rango de fechas
	FindByDateRange(ctx context.Context, startDate, endDate string, limit, offset int) ([]*sessionvideo.SessionVideo, error)
}
