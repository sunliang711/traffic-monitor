package dto

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

type RestoreStatusResp struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

type RestoreAdminPasswordReq struct {
	Username     string `json:"username" binding:"required"`
	RestoreToken string `json:"restore_token" binding:"required"`
	NewPassword  string `json:"new_password" binding:"required,min=6"`
}

type AdminProfileResp struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}
