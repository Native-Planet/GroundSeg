package common

// OwnerModule identifies the owning domain for a contract catalog entry.
type OwnerModule string

const (
	OwnerUploadService OwnerModule = "upload-domain"
	OwnerSystemWiFi    OwnerModule = "networking-domain"
	OwnerStartram      OwnerModule = "startram-domain"
)
