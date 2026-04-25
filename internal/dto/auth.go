package dto

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AdminProfileResp struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}
