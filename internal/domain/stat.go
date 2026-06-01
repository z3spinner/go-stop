package domain

type RouteStat struct {
	Origin      string
	Destination string
	Count       int
}

type ActivityCounts struct {
	AllTime   int `json:"all_time"`
	ThisYear  int `json:"this_year"`
	ThisMonth int `json:"this_month"`
}

type Stats struct {
	TopRoutes      []RouteStat    `json:"top_routes"`
	TotalConfirmed int            `json:"total_confirmed"`
	TotalThisWeek  int            `json:"total_this_week"`
	Searches       ActivityCounts `json:"searches"`
	RidesPosted    ActivityCounts `json:"rides_posted"`
}
