package videoservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unikyri/escritorio-remoto-backend/internal/application/actionlogservice"
	"github.com/unikyri/escritorio-remoto-backend/internal/application/interfaces"
	"github.com/unikyri/escritorio-remoto-backend/internal/domain/sessionvideo"
)

// VideoChunk representa un chunk de video recibido
type VideoChunk struct {
	SessionID   string `json:"session_id"`
	VideoID     string `json:"video_id"`
	ChunkData   []byte `json:"chunk_data"`
	IsLastChunk bool   `json:"is_last_chunk"`
	FileSize    int64  `json:"file_size"`
	Duration    int    `json:"duration"`
	FileName    string `json:"file_name"`
	ChunkIndex  int    `json:"chunk_index"`
}

// VideoUploadResult representa el resultado del procesamiento de chunks
type VideoUploadResult struct {
	IsComplete      bool    `json:"is_complete"`
	ChunksReceived  int     `json:"chunks_received"`
	TotalChunks     int     `json:"total_chunks"`
	ProgressPercent float64 `json:"progress_percent"`
	FilePath        string  `json:"file_path,omitempty"`
	Duration        int     `json:"duration,omitempty"`
	FileSize        int64   `json:"file_size,omitempty"`
}

// VideoUploadSession representa una sesión de subida en progreso
type VideoUploadSession struct {
	VideoID        string
	SessionID      string
	FileName       string
	FileSize       int64
	Duration       int
	Chunks         map[int][]byte
	TotalChunks    int
	ReceivedChunks int
	CreatedAt      time.Time
	LastChunkAt    time.Time
	mutex          sync.RWMutex
}

// VideoFrameInfo representa un frame individual de video para el nuevo sistema
type VideoFrameInfo struct {
	VideoID    string `json:"video_id"`
	SessionID  string `json:"session_id"`
	FrameIndex int    `json:"frame_index"`
	Timestamp  int64  `json:"timestamp"`
	FrameData  []byte `json:"frame_data"` // JPEG data
}

// VideoRecordingMetadata contiene los metadatos de una grabación de frames finalizada
type VideoRecordingMetadata struct {
	VideoID         string    `json:"video_id"`
	SessionID       string    `json:"session_id"`
	TotalFrames     int       `json:"total_frames"`
	FPS             float64   `json:"fps"`
	DurationSeconds float64   `json:"duration_seconds"`
	CompletedAt     time.Time `json:"completed_at"`
}

// IVideoService define la interfaz del servicio de video
type IVideoService interface {
	HandleUploadedVideoChunk(chunk VideoChunk) (*VideoUploadResult, error)
	FinalizeVideoUpload(ctx context.Context, sessionID, videoID, tempFilePath string, fileSizeMB float64, duration int) (*sessionvideo.SessionVideo, error)
	GetVideosBySessionID(ctx context.Context, sessionID string) ([]*sessionvideo.SessionVideo, error)
	GetVideoByID(ctx context.Context, videoID string) (*sessionvideo.SessionVideo, error)
	DeleteVideo(ctx context.Context, videoID string) error
	GetAllVideos(ctx context.Context, limit, offset int) ([]*sessionvideo.SessionVideo, error)

	// Nuevos métodos para el sistema de frames individuales
	SaveVideoFrame(frameInfo VideoFrameInfo) error
	FinalizeVideoRecording(recordingInfo VideoRecordingMetadata) error
}

// videoService implementa IVideoService
type videoService struct {
	videoRepository  interfaces.ISessionVideoRepository
	fileStorage      interfaces.IFileStorage
	actionLogService actionlogservice.IActionLogService

	// Mapa para tracking de uploads en progreso
	uploadSessions map[string]*VideoUploadSession
	uploadMutex    sync.RWMutex
}

// NewVideoService crea una nueva instancia del servicio de video
func NewVideoService(
	videoRepository interfaces.ISessionVideoRepository,
	fileStorage interfaces.IFileStorage,
	actionLogService actionlogservice.IActionLogService,
) IVideoService {
	return &videoService{
		videoRepository:  videoRepository,
		fileStorage:      fileStorage,
		actionLogService: actionLogService,
		uploadSessions:   make(map[string]*VideoUploadSession),
	}
}

// HandleUploadedVideoChunk maneja la recepción de chunks de video
func (vs *videoService) HandleUploadedVideoChunk(chunk VideoChunk) (*VideoUploadResult, error) {
	vs.uploadMutex.Lock()
	defer vs.uploadMutex.Unlock()

	// Decodificar chunk data si viene en base64
	var chunkData []byte

	// Si ChunkData contiene datos, usar directamente
	if len(chunk.ChunkData) > 0 {
		chunkData = chunk.ChunkData
	} else {
		return nil, fmt.Errorf("chunk data vacío para video %s", chunk.VideoID)
	}

	// Buscar o crear sesión de upload
	uploadSession, exists := vs.uploadSessions[chunk.VideoID]
	if !exists {
		// Calcular total de chunks basado en el tamaño del archivo
		chunkSize := 64 * 1024 // 64KB
		totalChunks := int((chunk.FileSize + int64(chunkSize) - 1) / int64(chunkSize))

		uploadSession = &VideoUploadSession{
			VideoID:     chunk.VideoID,
			SessionID:   chunk.SessionID,
			FileName:    chunk.FileName,
			FileSize:    chunk.FileSize,
			Duration:    chunk.Duration,
			Chunks:      make(map[int][]byte),
			TotalChunks: totalChunks,
			CreatedAt:   time.Now(),
			LastChunkAt: time.Now(),
		}
		vs.uploadSessions[chunk.VideoID] = uploadSession
	}

	uploadSession.mutex.Lock()
	defer uploadSession.mutex.Unlock()

	// Guardar chunk
	uploadSession.Chunks[chunk.ChunkIndex] = chunkData
	uploadSession.ReceivedChunks++
	uploadSession.LastChunkAt = time.Now()

	// Calcular progreso
	progressPercent := float64(uploadSession.ReceivedChunks) / float64(uploadSession.TotalChunks) * 100

	// Si es el último chunk o tenemos todos los chunks, procesar
	if chunk.IsLastChunk || uploadSession.ReceivedChunks >= uploadSession.TotalChunks {
		err := vs.processCompleteVideo(uploadSession)
		if err != nil {
			return nil, fmt.Errorf("error procesando video completo: %w", err)
		}

		// Generar ruta del archivo final
		fileName := fmt.Sprintf("%s_%s.mp4", uploadSession.SessionID, uploadSession.VideoID)
		finalPath := filepath.Join("videos", "processed", fileName)

		result := &VideoUploadResult{
			IsComplete:      true,
			ChunksReceived:  uploadSession.ReceivedChunks,
			TotalChunks:     uploadSession.TotalChunks,
			ProgressPercent: 100.0,
			FilePath:        finalPath,
			Duration:        uploadSession.Duration,
			FileSize:        uploadSession.FileSize,
		}

		// Limpiar sesión de upload
		delete(vs.uploadSessions, chunk.VideoID)

		return result, nil
	}

	// Retornar progreso parcial
	return &VideoUploadResult{
		IsComplete:      false,
		ChunksReceived:  uploadSession.ReceivedChunks,
		TotalChunks:     uploadSession.TotalChunks,
		ProgressPercent: progressPercent,
	}, nil
}

// processCompleteVideo ensambla todos los chunks y finaliza la subida
func (vs *videoService) processCompleteVideo(uploadSession *VideoUploadSession) error {
	// Ensamblar chunks en orden
	var completeVideo []byte
	for i := 0; i < uploadSession.TotalChunks; i++ {
		chunkData, exists := uploadSession.Chunks[i]
		if !exists {
			return fmt.Errorf("chunk %d faltante para video %s", i, uploadSession.VideoID)
		}
		completeVideo = append(completeVideo, chunkData...)
	}

	// Generar ruta de destino
	fileName := fmt.Sprintf("%s_%s.mp4", uploadSession.SessionID, uploadSession.VideoID)
	destinationPath := filepath.Join("videos", "processed", fileName)

	// Guardar archivo completo
	ctx := context.Background()
	finalPath, err := vs.fileStorage.SaveFile(ctx, destinationPath, completeVideo)
	if err != nil {
		return fmt.Errorf("error guardando video completo: %w", err)
	}

	// Calcular tamaño en MB
	fileSizeMB := float64(len(completeVideo)) / (1024 * 1024)

	// Finalizar upload
	_, err = vs.FinalizeVideoUpload(ctx, uploadSession.SessionID, uploadSession.VideoID, finalPath, fileSizeMB, uploadSession.Duration)
	if err != nil {
		return fmt.Errorf("error finalizando upload: %w", err)
	}

	return nil
}

// FinalizeVideoUpload mueve el video al almacenamiento final y actualiza la BD
func (vs *videoService) FinalizeVideoUpload(ctx context.Context, sessionID, videoID, tempFilePath string, fileSizeMB float64, duration int) (*sessionvideo.SessionVideo, error) {
	// Crear entidad SessionVideo
	video := sessionvideo.NewSessionVideoFromDB(
		videoID,
		tempFilePath,
		duration,
		time.Now(),
		sessionID,
		fileSizeMB,
		time.Now(),
		time.Now(),
	)

	// Guardar en base de datos
	err := vs.videoRepository.Save(ctx, video)
	if err != nil {
		return nil, fmt.Errorf("error guardando video en BD: %w", err)
	}

	// Registrar en audit log
	entityType := "SESSION_VIDEO"
	err = vs.actionLogService.LogAction(ctx, "VIDEO_UPLOADED",
		fmt.Sprintf("Video de sesión subido exitosamente - Archivo: %s", filepath.Base(tempFilePath)),
		"admin-000-000-000-000000000001", // TODO: obtener del contexto
		&videoID,
		&entityType,
		map[string]interface{}{
			"video_id":     videoID,
			"session_id":   sessionID,
			"file_size_mb": fileSizeMB,
			"duration":     duration,
			"file_path":    tempFilePath,
		})
	if err != nil {
		// Log pero no fallar
		fmt.Printf("Warning: error registrando audit log para video %s: %v\n", videoID, err)
	}

	return video, nil
}

// GetVideosBySessionID obtiene todos los videos de una sesión
func (vs *videoService) GetVideosBySessionID(ctx context.Context, sessionID string) ([]*sessionvideo.SessionVideo, error) {
	return vs.videoRepository.FindBySessionID(ctx, sessionID)
}

// GetVideoByID obtiene un video por su ID
func (vs *videoService) GetVideoByID(ctx context.Context, videoID string) (*sessionvideo.SessionVideo, error) {
	return vs.videoRepository.FindByID(ctx, videoID)
}

// DeleteVideo elimina un video
func (vs *videoService) DeleteVideo(ctx context.Context, videoID string) error {
	// Buscar video para obtener ruta del archivo
	video, err := vs.videoRepository.FindByID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("error buscando video: %w", err)
	}

	// Eliminar archivo físico
	err = vs.fileStorage.DeleteFile(ctx, video.FilePath())
	if err != nil {
		// Log pero continuar con eliminación de BD
		fmt.Printf("Warning: error eliminando archivo físico %s: %v\n", video.FilePath(), err)
	}

	// Eliminar de BD
	err = vs.videoRepository.Delete(ctx, videoID)
	if err != nil {
		return fmt.Errorf("error eliminando video de BD: %w", err)
	}

	return nil
}

// GetAllVideos obtiene todos los videos con paginación
func (vs *videoService) GetAllVideos(ctx context.Context, limit, offset int) ([]*sessionvideo.SessionVideo, error) {
	return vs.videoRepository.FindAll(ctx, limit, offset)
}

// SaveVideoFrame guarda un frame individual de video
func (vs *videoService) SaveVideoFrame(frameInfo VideoFrameInfo) error {
	// Crear directorio para los frames de este video si no existe
	framesDir := filepath.Join("storage", "session_videos", frameInfo.VideoID, "frames")
	err := os.MkdirAll(framesDir, 0755)
	if err != nil {
		return fmt.Errorf("error creando directorio de frames: %w", err)
	}

	// Generar nombre del archivo con padding para ordenamiento correcto
	// frame_000001.jpg, frame_000002.jpg, etc.
	frameFileName := fmt.Sprintf("frame_%06d.jpg", frameInfo.FrameIndex)
	frameFilePath := filepath.Join(framesDir, frameFileName)

	// Guardar frame como archivo JPEG
	err = os.WriteFile(frameFilePath, frameInfo.FrameData, 0644)
	if err != nil {
		return fmt.Errorf("error guardando frame %d: %w", frameInfo.FrameIndex, err)
	}

	return nil
}

// FinalizeVideoRecording finaliza una grabación de frames
func (vs *videoService) FinalizeVideoRecording(recordingInfo VideoRecordingMetadata) error {
	// Construir la ruta base donde están guardados los frames
	framesBasePath := filepath.Join("storage", "session_videos", recordingInfo.VideoID, "frames")

	// Verificar que el directorio de frames existe
	if _, err := os.Stat(framesBasePath); os.IsNotExist(err) {
		return fmt.Errorf("directorio de frames no encontrado: %s", framesBasePath)
	}

	// Calcular tamaño total aproximado de todos los frames
	totalSizeMB := vs.calculateFramesDirSize(framesBasePath)

	// Crear entidad SessionVideo con los metadatos de la grabación de frames
	video := sessionvideo.NewSessionVideoFromDB(
		recordingInfo.VideoID,
		framesBasePath,                     // Usar directorio base en lugar de archivo MP4
		int(recordingInfo.DurationSeconds), // Duración en segundos
		recordingInfo.CompletedAt,          // Momento de finalización
		recordingInfo.SessionID,            // ID de la sesión
		totalSizeMB,                        // Tamaño total de frames
		recordingInfo.CompletedAt,          // created_at
		recordingInfo.CompletedAt,          // updated_at
	)

	// Guardar en base de datos
	ctx := context.Background()
	err := vs.videoRepository.Save(ctx, video)
	if err != nil {
		return fmt.Errorf("error guardando metadatos de video en BD: %w", err)
	}

	// Registrar en audit log
	entityType := "SESSION_VIDEO"
	err = vs.actionLogService.LogAction(ctx, "VIDEO_RECORDING_ENDED",
		fmt.Sprintf("Grabación de frames finalizada - VideoID: %s, Frames: %d, FPS: %.2f",
			recordingInfo.VideoID, recordingInfo.TotalFrames, recordingInfo.FPS),
		"admin-000-000-000-000000000001", // TODO: obtener del contexto
		&recordingInfo.VideoID,
		&entityType,
		map[string]interface{}{
			"video_id":         recordingInfo.VideoID,
			"session_id":       recordingInfo.SessionID,
			"total_frames":     recordingInfo.TotalFrames,
			"fps":              recordingInfo.FPS,
			"duration_seconds": recordingInfo.DurationSeconds,
			"frames_dir":       framesBasePath,
			"total_size_mb":    totalSizeMB,
		})
	if err != nil {
		// Log pero no fallar
		fmt.Printf("Warning: error registrando audit log para finalización de grabación %s: %v\n", recordingInfo.VideoID, err)
	}

	return nil
}

// calculateFramesDirSize calcula el tamaño total de un directorio de frames en MB
func (vs *videoService) calculateFramesDirSize(dirPath string) float64 {
	var totalSize int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		// Si hay error, retornar 0
		return 0.0
	}

	// Convertir a MB
	return float64(totalSize) / (1024 * 1024)
}
