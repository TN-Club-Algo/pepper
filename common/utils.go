package common

const (
	RestPort      int32  = 8080
	InputEndpoint string = "/input"
	InitEndPoint  string = "/init"
)

func SliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
