package hint

import "encoding/json"

// Error is a structured response indicating a non-2xx HTTP response.
type Error struct {
	HTTPStatusCode int    `json:"status"`
	Message        string `json:"message"`
	Errors         struct {
		Location string `json:"location"`
		Type     string `json:"type"`
		ID       string `json:"id"`
	} `json:"errors"`
}

func (e *Error) Error() string {

	data, _ := json.Marshal(e)
	return string(data)
}
