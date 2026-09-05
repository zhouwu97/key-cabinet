package dto

type WechatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

type UpdateProfileRequest struct {
	Name       string `json:"name"`
	StudentNo  string `json:"studentNo"`
	Department string `json:"department"`
	Phone      string `json:"phone"`
}
