package k8s

// TODO: Define ObjectMeta struct
type ObjectMeta struct {
	Name string `json:"name"`
	// TODO: add other fields
}

// TODO: Define list response struct
type ListResponse struct {
	Items []interface{} `json:"items"`
	// TODO: add metadata
}

// TODO: Helper functions for JSON responses
func NewListResponse(items []interface{}) ListResponse {
	// TODO
	return ListResponse{}
}