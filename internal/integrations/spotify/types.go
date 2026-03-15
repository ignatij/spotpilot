package spotify

// API response types — kept internal, never leaked across the package boundary.

type searchResponse struct {
	Tracks  trackPage  `json:"tracks"`
	Albums  albumPage  `json:"albums"`
	Artists artistPage `json:"artists"`
}

type trackPage struct {
	Items []trackItem `json:"items"`
}

type albumPage struct {
	Items []albumItem `json:"items"`
}

type artistPage struct {
	Items []artistItem `json:"items"`
}

type trackItem struct {
	URI     string       `json:"uri"`
	Name    string       `json:"name"`
	Artists []artistItem `json:"artists"`
	Album   struct {
		Name string `json:"name"`
	} `json:"album"`
}

type albumItem struct {
	URI     string       `json:"uri"`
	Name    string       `json:"name"`
	Artists []artistItem `json:"artists"`
}

type artistItem struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type playerResponse struct {
	IsPlaying bool       `json:"is_playing"`
	Device    deviceItem `json:"device"`
	Item      trackItem  `json:"item"`
}

type devicesResponse struct {
	Devices []deviceItem `json:"devices"`
}

type deviceItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsActive bool   `json:"is_active"`
}
