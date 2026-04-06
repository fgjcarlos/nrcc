package model

type LibraryPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Direct  bool   `json:"direct"`
}

type LibraryList struct {
	Items []LibraryPackage `json:"items"`
}

type LibraryOperationResult struct {
	Package   LibraryPackage `json:"package"`
	Message   string         `json:"message"`
	Output    string         `json:"output,omitempty"`
	Operation string         `json:"operation"`
}

type OperationStatus struct {
	Busy      bool   `json:"busy"`
	Type      string `json:"type,omitempty"`
	Detail    string `json:"detail,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
}
