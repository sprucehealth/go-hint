package hint

type Membership struct {
	StartDate string   `json:"start_date"`
	EndDate   *string  `json:"end_date"`
	Status    string   `json:"status"`
	Plan      *Plan    `json:"plan"`
	Company   *Company `json:"company"`
}

type Plan struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Company struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
