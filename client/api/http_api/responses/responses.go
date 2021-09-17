package responses

type BaseResponse struct {
	ErrorMessage string      `json:"error_message,omitempty"`
	Result       interface{} `json:"result"`
}
