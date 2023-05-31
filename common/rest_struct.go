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

type VmInput struct {
	ID    string `json:"id"`
	Input string `json:"input"`
}

type InnerTestResult struct {
	ProblemName string `json:"problemName"`
	ID          string `json:"id"`
	Index       int    `json:"index"`
	Answer      string `json:"answer"`
	Ok          bool   `json:"ok"`
}

type TestResult struct {
	ID string `json:"id"`
	Ok bool   `json:"ok"`
}

type TestRequest struct {
	ProblemName     string               `json:"problemName"`
	ID              string               `json:"id"`              // test id
	ProgramLocation string               `json:"programLocation"` // user program location
	UserProgram     string               `json:"userProgram"`     // user main program
	TestType        string               `json:"testType"`
	TestCount       int                  `json:"testCount"`
	Tests           InnerInputOutputTest `json:"tests"`
}

type InnerInputOutputTest struct {
	Inputs  []string `json:"input"`
	Outputs []string `json:"output"`
}

type VmInit struct {
	ProgramType string `json:"programType"`
	UserProgram string `json:"userProgram"`
	IsDirectory bool   `json:"isDirectory"`
	TestType    string `json:"testType"`
	TestCount   int    `json:"testCount"`
}
