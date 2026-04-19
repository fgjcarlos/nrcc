package model

// Asset represents an uploaded branding asset (favicon, header image, login image).
type Asset struct {
	ID        string `json:"id"`
	Category  string `json:"category"`  // "favicon", "header", "login"
	Filename  string `json:"filename"`  // stored filename (unique)
	Original  string `json:"original"`  // original upload filename
	MIMEType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	URL       string `json:"url"` // serving URL: /assets/{category}/{filename}
	CreatedAt string `json:"createdAt"`
}

// AssetList wraps a slice of assets for API responses.
type AssetList struct {
	Items []Asset `json:"items"`
}
