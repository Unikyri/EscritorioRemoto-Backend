package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/sessionvideo"
)

// sessionVideoRepository implementa ISessionVideoRepository
type sessionVideoRepository struct {
	db *sql.DB
}

// NewSessionVideoRepository crea una nueva instancia del repositorio
func NewSessionVideoRepository(db *sql.DB) interfaces.ISessionVideoRepository {
	return &sessionVideoRepository{
		db: db,
	}
}

// Save guarda un nuevo video de sesión
func (r *sessionVideoRepository) Save(ctx context.Context, video *sessionvideo.SessionVideo) error {
	query := `
		INSERT INTO session_videos (
			video_id, file_path, duration_seconds, recorded_at, 
			associated_session_id, file_size_mb, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		video.VideoID(),
		video.FilePath(),
		video.DurationSeconds(),
		video.RecordedAt(),
		video.AssociatedSessionID(),
		video.FileSizeMB(),
		video.CreatedAt(),
		video.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("error guardando video en BD: %w", err)
	}

	return nil
}

// FindByID busca un video por su ID
func (r *sessionVideoRepository) FindByID(ctx context.Context, videoID string) (*sessionvideo.SessionVideo, error) {
	query := `
		SELECT video_id, file_path, duration_seconds, recorded_at, 
			   associated_session_id, file_size_mb, created_at, updated_at
		FROM session_videos
		WHERE video_id = ?
	`

	row := r.db.QueryRowContext(ctx, query, videoID)

	var id, filePath, sessionID string
	var duration int
	var recordedAt, createdAt, updatedAt sql.NullTime
	var fileSizeMB float64

	err := row.Scan(&id, &filePath, &duration, &recordedAt, &sessionID, &fileSizeMB, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("video no encontrado: %s", videoID)
		}
		return nil, fmt.Errorf("error buscando video: %w", err)
	}

	return sessionvideo.NewSessionVideoFromDB(
		id,
		filePath,
		duration,
		recordedAt.Time,
		sessionID,
		fileSizeMB,
		createdAt.Time,
		updatedAt.Time,
	), nil
}

// FindBySessionID busca videos asociados a una sesión específica
func (r *sessionVideoRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*sessionvideo.SessionVideo, error) {
	query := `
		SELECT video_id, file_path, duration_seconds, recorded_at, 
			   associated_session_id, file_size_mb, created_at, updated_at
		FROM session_videos
		WHERE associated_session_id = ?
		ORDER BY recorded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("error buscando videos por sesión: %w", err)
	}
	defer rows.Close()

	var videos []*sessionvideo.SessionVideo

	for rows.Next() {
		var id, filePath, assocSessionID string
		var duration int
		var recordedAt, createdAt, updatedAt sql.NullTime
		var fileSizeMB float64

		err := rows.Scan(&id, &filePath, &duration, &recordedAt, &assocSessionID, &fileSizeMB, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("error escaneando video: %w", err)
		}

		video := sessionvideo.NewSessionVideoFromDB(
			id,
			filePath,
			duration,
			recordedAt.Time,
			assocSessionID,
			fileSizeMB,
			createdAt.Time,
			updatedAt.Time,
		)

		videos = append(videos, video)
	}

	return videos, nil
}

// Update actualiza un video existente
func (r *sessionVideoRepository) Update(ctx context.Context, video *sessionvideo.SessionVideo) error {
	query := `
		UPDATE session_videos 
		SET file_path = ?, duration_seconds = ?, file_size_mb = ?, updated_at = ?
		WHERE video_id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		video.FilePath(),
		video.DurationSeconds(),
		video.FileSizeMB(),
		video.UpdatedAt(),
		video.VideoID(),
	)

	if err != nil {
		return fmt.Errorf("error actualizando video: %w", err)
	}

	return nil
}

// Delete elimina un video por su ID
func (r *sessionVideoRepository) Delete(ctx context.Context, videoID string) error {
	query := `DELETE FROM session_videos WHERE video_id = ?`

	_, err := r.db.ExecContext(ctx, query, videoID)
	if err != nil {
		return fmt.Errorf("error eliminando video: %w", err)
	}

	return nil
}

// FindAll obtiene todos los videos con paginación
func (r *sessionVideoRepository) FindAll(ctx context.Context, limit, offset int) ([]*sessionvideo.SessionVideo, error) {
	query := `
		SELECT video_id, file_path, duration_seconds, recorded_at, 
			   associated_session_id, file_size_mb, created_at, updated_at
		FROM session_videos
		ORDER BY recorded_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo videos: %w", err)
	}
	defer rows.Close()

	var videos []*sessionvideo.SessionVideo

	for rows.Next() {
		var id, filePath, sessionID string
		var duration int
		var recordedAt, createdAt, updatedAt sql.NullTime
		var fileSizeMB float64

		err := rows.Scan(&id, &filePath, &duration, &recordedAt, &sessionID, &fileSizeMB, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("error escaneando video: %w", err)
		}

		video := sessionvideo.NewSessionVideoFromDB(
			id,
			filePath,
			duration,
			recordedAt.Time,
			sessionID,
			fileSizeMB,
			createdAt.Time,
			updatedAt.Time,
		)

		videos = append(videos, video)
	}

	return videos, nil
}

// Count obtiene el total de videos
func (r *sessionVideoRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM session_videos`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error contando videos: %w", err)
	}

	return count, nil
}

// FindByDateRange busca videos en un rango de fechas
func (r *sessionVideoRepository) FindByDateRange(ctx context.Context, startDate, endDate string, limit, offset int) ([]*sessionvideo.SessionVideo, error) {
	query := `
		SELECT video_id, file_path, duration_seconds, recorded_at, 
			   associated_session_id, file_size_mb, created_at, updated_at
		FROM session_videos
		WHERE recorded_at BETWEEN ? AND ?
		ORDER BY recorded_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error buscando videos por rango de fechas: %w", err)
	}
	defer rows.Close()

	var videos []*sessionvideo.SessionVideo

	for rows.Next() {
		var id, filePath, sessionID string
		var duration int
		var recordedAt, createdAt, updatedAt sql.NullTime
		var fileSizeMB float64

		err := rows.Scan(&id, &filePath, &duration, &recordedAt, &sessionID, &fileSizeMB, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("error escaneando video: %w", err)
		}

		video := sessionvideo.NewSessionVideoFromDB(
			id,
			filePath,
			duration,
			recordedAt.Time,
			sessionID,
			fileSizeMB,
			createdAt.Time,
			updatedAt.Time,
		)

		videos = append(videos, video)
	}

	return videos, nil
}
