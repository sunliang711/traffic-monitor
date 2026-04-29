package dto

type GuestModeResp struct {
	Enabled bool `json:"enabled"`
}

type UpdateGuestModeReq struct {
	Enabled bool `json:"enabled"`
}
