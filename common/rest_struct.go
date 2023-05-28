package common

const (
	JAVA   string = "java"
	CPP    string = "cpp"
	PYTHON string = "python"
	C      string = "c"
)

const (
	TestTypeInputOutput string = "input/output"
)

type Test struct {
	ID          string `json:"id"`          // test id
	UserProgram string `json:"userProgram"` // user program location
}

type VmInput struct {
	ID    string `json:"id"`
	Input string `json:"input"`
}

type VmInit struct {
	ProgramType string `json:"programType"`
	UserProgram string `json:"userProgram"`
	IsDirectory bool   `json:"isDirectory"`
	TestType    string `json:"testType"`
	TestCount   int    `json:"testCount"`
}
