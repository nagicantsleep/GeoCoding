package models

// Location represents a single addressable point, containing its decomposed Japanese address components and its precise geographic coordinates.
type Location struct {
	ID         int     `json:"id"`
	Prefecture string  `json:"prefecture"`
	Municipality string `json:"municipality"`
	Address1   string  `json:"address1"`
	Address2   string  `json:"address2"`
	BlockLot   string  `json:"block_lot"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}