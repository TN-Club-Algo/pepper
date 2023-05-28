package common

const (
	RestPort      int64  = 8080
	InputEndpoint string = "/input"
	InitEndPoint  string = "/init"
	PingEndPoint  string = "/ping"
)

func SliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
