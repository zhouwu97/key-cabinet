package dto

type MatchRoomQuery struct {
	DeviceID string `form:"deviceId"`
	RoomNo   string `form:"roomNo" binding:"required"`
}

type MatchedKeyDTO struct {
	KeyID        string `json:"keyId"`
	KeyName      string `json:"keyName"`
	RoomNo       string `json:"roomNo"`
	Building     string `json:"building,omitempty"`
	DeviceID     string `json:"deviceId"`
	SlotID       string `json:"slotId"`
	SlotNo       int    `json:"slotNo"`
	Status       string `json:"status"`
	SlotPresence string `json:"slotPresence"`
	IsBorrowable bool   `json:"isBorrowable"`
}

type CabinetDirectDispenseRequest struct {
	RequestID string `json:"requestId" binding:"required"`
	DeviceID  string `json:"deviceId"`
	RoomNo    string `json:"roomNo"`
	KeyID     string `json:"keyId"`
	StudentNo string `json:"studentNo,omitempty"`
	Purpose   string `json:"purpose"`
}

type CabinetDirectDispenseResponse struct {
	OperationID    string `json:"operationId"`
	BorrowRecordID string `json:"borrowRecordId"`
	KeyID          string `json:"keyId"`
	KeyName        string `json:"keyName"`
	RoomNo         string `json:"roomNo"`
	DeviceID       string `json:"deviceId"`
	SlotID         string `json:"slotId"`
	SlotNo         int    `json:"slotNo"`
	Status         string `json:"status"`
}

type FaceAuthRequest struct {
	DeviceID       string  `json:"deviceId"`
	StudentNo      string  `json:"studentNo" binding:"required"`
	Confidence     float64 `json:"confidence"`
	LivenessPassed bool    `json:"livenessPassed"`
}

type FaceAuthResponse struct {
	User               interface{} `json:"user"`
	ActiveReservations interface{} `json:"activeReservations"`
	ActiveBorrows      interface{} `json:"activeBorrows"`
	FaceSessionToken   string      `json:"faceSessionToken"`
	CabinetToken       string      `json:"cabinetToken"`
}
