package sessionvideo

import (
	"time"

	"github.com/google/uuid"
)

// SessionVideo representa un video de sesión de control remoto
type SessionVideo struct {
	videoID             string
	filePath            string
	durationSeconds     int
	recordedAt          time.Time
	associatedSessionID string
	fileSizeMB          float64
	createdAt           time.Time
	updatedAt           time.Time
}

// NewSessionVideo crea una nueva instancia de SessionVideo
func NewSessionVideo(
	filePath string,
	durationSeconds int,
	associatedSessionID string,
	fileSizeMB float64,
) *SessionVideo {
	return &SessionVideo{
		videoID:             uuid.New().String(),
		filePath:            filePath,
		durationSeconds:     durationSeconds,
		recordedAt:          time.Now(),
		associatedSessionID: associatedSessionID,
		fileSizeMB:          fileSizeMB,
		createdAt:           time.Now(),
		updatedAt:           time.Now(),
	}
}

// NewSessionVideoFromDB reconstruye SessionVideo desde base de datos
func NewSessionVideoFromDB(
	videoID string,
	filePath string,
	durationSeconds int,
	recordedAt time.Time,
	associatedSessionID string,
	fileSizeMB float64,
	createdAt time.Time,
	updatedAt time.Time,
) *SessionVideo {
	return &SessionVideo{
		videoID:             videoID,
		filePath:            filePath,
		durationSeconds:     durationSeconds,
		recordedAt:          recordedAt,
		associatedSessionID: associatedSessionID,
		fileSizeMB:          fileSizeMB,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}
}

// Getters
func (sv *SessionVideo) VideoID() string {
	return sv.videoID
}

func (sv *SessionVideo) FilePath() string {
	return sv.filePath
}

func (sv *SessionVideo) DurationSeconds() int {
	return sv.durationSeconds
}

func (sv *SessionVideo) RecordedAt() time.Time {
	return sv.recordedAt
}

func (sv *SessionVideo) AssociatedSessionID() string {
	return sv.associatedSessionID
}

func (sv *SessionVideo) FileSizeMB() float64 {
	return sv.fileSizeMB
}

func (sv *SessionVideo) CreatedAt() time.Time {
	return sv.createdAt
}

func (sv *SessionVideo) UpdatedAt() time.Time {
	return sv.updatedAt
}

// SetFilePath actualiza la ruta del archivo (para cuando se mueve al almacenamiento final)
func (sv *SessionVideo) SetFilePath(filePath string) {
	sv.filePath = filePath
	sv.updatedAt = time.Now()
}

// SetFileSizeMB actualiza el tamaño del archivo
func (sv *SessionVideo) SetFileSizeMB(fileSizeMB float64) {
	sv.fileSizeMB = fileSizeMB
	sv.updatedAt = time.Now()
}

// IsValid verifica si el SessionVideo es válido
func (sv *SessionVideo) IsValid() bool {
	return sv.videoID != "" &&
		sv.filePath != "" &&
		sv.durationSeconds >= 0 &&
		sv.associatedSessionID != "" &&
		sv.fileSizeMB >= 0
}
