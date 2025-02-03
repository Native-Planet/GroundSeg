package structs

// version server payload root struct
type Version struct {
	Groundseg map[string]Channel `json:"groundseg"`
}

// version server payload substruct
type Channel struct {
	Groundseg VersionDetails `json:"groundseg"`
	Manual    VersionDetails `json:"manual"`
	Minio     VersionDetails `json:"minio"`
	Miniomc   VersionDetails `json:"miniomc"`
	Netdata   VersionDetails `json:"netdata"`
	Vere      VersionDetails `json:"vere"`
	Webui     VersionDetails `json:"webui"`
	Wireguard VersionDetails `json:"wireguard"`
}

// version server payload substruct
type VersionDetails struct {
	Amd64Sha256 string `json:"amd64_sha256"`
	Amd64URL    string `json:"amd64_url,omitempty"`
	Arm64Sha256 string `json:"arm64_sha256"`
	Arm64URL    string `json:"arm64_url,omitempty"`
	SlsaURL     string `json:"slsa_url,omitempty"`
	Major       int    `json:"major,omitempty"`
	Minor       int    `json:"minor,omitempty"`
	Patch       int    `json:"patch,omitempty"`
	Repo        string `json:"repo,omitempty"`
	Tag         string `json:"tag,omitempty"`
}
