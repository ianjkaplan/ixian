package types

// CreatePetRequest
type CreatePetRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

// Error
type Error struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// Owner
type Owner struct {
	Email string `json:"email,omitempty"`
	Id    string `json:"id"`
	Name  string `json:"name"`
}

// Pet
type Pet struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Owner  any    `json:"owner,omitempty"`
	Status string `json:"status,omitempty"`
	Tag    string `json:"tag,omitempty"`
}
