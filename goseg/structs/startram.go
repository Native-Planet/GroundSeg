package structs

// blob from /retrieve path
type StartramRetrieve struct {
	Action     string `json:"action"`
	Conf       string `json:"conf"`
	Debug      any    `json:"debug"`
	Error      int    `json:"error"`
	Lease      string `json:"lease"`
	Ongoing    int    `json:"ongoing"`
	Pubkey     string `json:"pubkey"`
	Region     string `json:"region"`
	Status     string `json:"status"`
	Subdomains []struct {
		Alias   string `json:"alias"`
		Port    int    `json:"port"`
		Status  string `json:"status"`
		SvcType string `json:"svc_type"`
		URL     string `json:"url"`
	} `json:"subdomains"`
}

// startram region server subobject
type StartramRegion struct {
	Country string `json:"country"`
	Desc    string `json:"desc"`
}
