package dto

// WebSocket Message Types
const (
	MessageTypeClientAuth         = "CLIENT_AUTH_REQUEST"
	MessageTypeClientAuthResp     = "CLIENT_AUTH_RESPONSE"
	MessageTypePCRegistration     = "PC_REGISTRATION_REQUEST"
	MessageTypePCRegistrationResp = "PC_REGISTRATION_RESPONSE"
	MessageTypeHeartbeat          = "HEARTBEAT"
	MessageTypeHeartbeatResp      = "HEARTBEAT_RESPONSE"

	// Remote Control Streaming Messages
	MessageTypeScreenFrame  = "screen_frame"
	MessageTypeInputCommand = "input_command"
)

// Base message structure
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Client Authentication Messages
type ClientAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ClientAuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	UserID  string `json:"userId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PC Registration Messages
type PCRegistrationRequest struct {
	PCIdentifier string `json:"pcIdentifier"`
	IP           string `json:"ip,omitempty"` // Optional, can be detected from connection
}

type PCRegistrationResponse struct {
	Success bool   `json:"success"`
	PCID    string `json:"pcId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Heartbeat Messages
type HeartbeatRequest struct {
	Timestamp int64 `json:"timestamp"`
}

type HeartbeatResponse struct {
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"`
}

// Screen Streaming Messages
// ScreenFrame represents a captured screen frame from client
type ScreenFrame struct {
	SessionID   string `json:"session_id"`
	Timestamp   int64  `json:"timestamp"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Format      string `json:"format"`            // "jpeg", "png", etc.
	Quality     int    `json:"quality,omitempty"` // For JPEG compression (1-100)
	FrameData   []byte `json:"frame_data"`        // Image bytes
	SequenceNum int64  `json:"sequence_num"`
}

// InputCommand represents a remote input command (mouse/keyboard) from admin
type InputCommand struct {
	SessionID string                 `json:"session_id"`
	Timestamp int64                  `json:"timestamp"`
	EventType string                 `json:"event_type"` // "mouse", "keyboard"
	Action    string                 `json:"action"`     // "move", "click", "scroll", "keydown", "keyup", "type"
	Payload   map[string]interface{} `json:"payload"`    // Event-specific data
}

// Mouse Event Payload Fields (for reference)
type MouseEventPayload struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button,omitempty"` // "left", "right", "middle"
	Delta  int    `json:"delta,omitempty"`  // For scroll events
}

// Keyboard Event Payload Fields (for reference)
type KeyboardEventPayload struct {
	Key       string   `json:"key"`                 // Key identifier
	Code      string   `json:"code,omitempty"`      // Physical key code
	Text      string   `json:"text,omitempty"`      // For typing text
	Modifiers []string `json:"modifiers,omitempty"` // ["ctrl", "alt", "shift", "meta"]
}
