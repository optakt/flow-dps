package object

// Error is used to return rich errors from the API instead of utilizing HTTP
// status codes (which often do not have a good analog). Both the code and the
// message fields can be individually used to correctly identify an error.
// Implementations must use unique values for both fields.
//
// Example for detail fields given in the Rosetta API documentation are
// `address` and `error`.
type Error struct {
	Code        uint                   `json:"code"`
	Message     string                 `json:"message"`
	Description string                 `json:"description"`
	Retriable   bool                   `json:"retriable"`
	Details     map[string]interface{} `json:"details"`
}
