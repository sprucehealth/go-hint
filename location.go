package hint

type Location struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	LocationGroup *LocationGroup `json:"location_group,omitempty"`
}

type LocationGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
