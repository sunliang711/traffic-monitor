package dto

type Response struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type HealthResp struct {
	AppName string `json:"app_name"`
	DB      string `json:"db"`
	Env     string `json:"env"`
	Status  string `json:"status"`
}
