package validation

type CreateLocation struct {
	Name       string  `json:"name" validate:"required"`
	Department string  `json:"department" validate:"required"`
	Latitude   float64 `json:"latitude" validate:"required"`
	Longitude  float64 `json:"longitude" validate:"required"`
	Radius     float64 `json:"radius" validate:"required,gt=0"`
}

type UpdateLocation struct {
	Name       string  `json:"name" validate:"required"`
	Department string  `json:"department" validate:"required"`
	Latitude   float64 `json:"latitude" validate:"required"`
	Longitude  float64 `json:"longitude" validate:"required"`
	Radius     float64 `json:"radius" validate:"required,gt=0"`
}

type QueryLocation struct {
	Page   int    `validate:"omitempty,number"`
	Limit  int    `validate:"omitempty,number"`
	Search string `validate:"omitempty"`
}
