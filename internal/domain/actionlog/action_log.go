package actionlog

import (
	"time"
)

// ActionType define los tipos de acciones que se pueden registrar
type ActionType string

const (
	ActionUserLogin             ActionType = "USER_LOGIN"
	ActionUserLogout            ActionType = "USER_LOGOUT"
	ActionUserCreated           ActionType = "USER_CREATED"
	ActionPCRegistered          ActionType = "PC_REGISTERED"
	ActionPCStatusChanged       ActionType = "PC_STATUS_CHANGED"
	ActionRemoteSessionStarted  ActionType = "REMOTE_SESSION_STARTED"
	ActionRemoteSessionEnded    ActionType = "REMOTE_SESSION_ENDED"
	ActionFileTransferInitiated ActionType = "FILE_TRANSFER_INITIATED"
	ActionFileTransferCompleted ActionType = "FILE_TRANSFER_COMPLETED"
	ActionFileTransferFailed    ActionType = "FILE_TRANSFER_FAILED"
	ActionVideoRecordingStarted ActionType = "VIDEO_RECORDING_STARTED"
	ActionVideoRecordingEnded   ActionType = "VIDEO_RECORDING_ENDED"
	ActionVideoUploaded         ActionType = "VIDEO_UPLOADED"
)

// ActionLog representa una entrada en el log de auditoría
type ActionLog struct {
	logID             int64
	timestamp         time.Time
	actionType        ActionType
	description       string
	performedByUserID string
	subjectEntityID   *string
	subjectEntityType *string
	details           map[string]interface{}
	createdAt         time.Time
}

// NewActionLog crea una nueva entrada de log de auditoría
func NewActionLog(
	actionType ActionType,
	description string,
	performedByUserID string,
	subjectEntityID *string,
	subjectEntityType *string,
	details map[string]interface{},
) *ActionLog {
	now := time.Now().UTC()

	return &ActionLog{
		logID:             0, // Se asignará por la base de datos (AUTO_INCREMENT)
		timestamp:         now,
		actionType:        actionType,
		description:       description,
		performedByUserID: performedByUserID,
		subjectEntityID:   subjectEntityID,
		subjectEntityType: subjectEntityType,
		details:           details,
		createdAt:         now,
	}
}

// NewActionLogFromDB crea un ActionLog desde datos de base de datos
func NewActionLogFromDB(
	logID int64,
	timestamp time.Time,
	actionType ActionType,
	description string,
	performedByUserID string,
	subjectEntityID *string,
	subjectEntityType *string,
	details map[string]interface{},
	createdAt time.Time,
) *ActionLog {
	return &ActionLog{
		logID:             logID,
		timestamp:         timestamp,
		actionType:        actionType,
		description:       description,
		performedByUserID: performedByUserID,
		subjectEntityID:   subjectEntityID,
		subjectEntityType: subjectEntityType,
		details:           details,
		createdAt:         createdAt,
	}
}

// Getters
func (al *ActionLog) LogID() int64 {
	return al.logID
}

func (al *ActionLog) Timestamp() time.Time {
	return al.timestamp
}

func (al *ActionLog) ActionType() ActionType {
	return al.actionType
}

func (al *ActionLog) Description() string {
	return al.description
}

func (al *ActionLog) PerformedByUserID() string {
	return al.performedByUserID
}

func (al *ActionLog) SubjectEntityID() *string {
	return al.subjectEntityID
}

func (al *ActionLog) SubjectEntityType() *string {
	return al.subjectEntityType
}

func (al *ActionLog) Details() map[string]interface{} {
	return al.details
}

func (al *ActionLog) CreatedAt() time.Time {
	return al.createdAt
}

// SetLogID asigna el ID del log (usado por el repositorio después de guardar)
func (al *ActionLog) SetLogID(logID int64) {
	al.logID = logID
}
