package core

type HttpTarget struct {
	Path        string `json:"path"`
	Host        string `json:"host"`
	StripPrefix bool   `json:"stripPrefix"`
}
