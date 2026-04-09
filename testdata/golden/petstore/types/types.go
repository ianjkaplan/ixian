package types

// CreatePetRequest
type CreatePetRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

// Error — Error response
type Error struct {
	// Error code
	Code int32 `json:"code"`
	// Human-readable error message
	Message string `json:"message"`
}

// Owner
type Owner struct {
	Email string `json:"email,omitempty"`
	Id    string `json:"id"`
	Name  string `json:"name"`
}

// Pet — A pet in the store
type Pet struct {
	// Unique identifier for the pet
	Id string `json:"id"`
	// Name of the pet
	Name   string `json:"name"`
	Owner  any    `json:"owner,omitempty"`
	Status string `json:"status,omitempty"`
	// Optional tag for categorization
	Tag string `json:"tag,omitempty"`
}
