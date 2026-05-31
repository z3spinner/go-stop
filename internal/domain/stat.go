package domain

type RouteStat struct {
	Origin      string
	Destination string
	Count       int
}

type Stats struct {
	TopRoutes      []RouteStat `json:"top_routes"`
	TotalConfirmed int         `json:"total_confirmed"`
	TotalThisWeek  int         `json:"total_this_week"`
}
