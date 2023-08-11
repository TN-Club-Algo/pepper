package common

const (
	JAVA   string = "java"
	CPP    string = "c_cpp"
	PYTHON string = "python"
	C      string = "c"
	KOTLIN string = "kotlin"
)

const (
	TestTypeInputOutput string = "INPUT_OUTPUT"
)

type VmInput struct {
	ID    string `json:"id"`
	Input string `json:"input"`
	Type  string `json:"type"`
}

type InnerTestResult struct {
	ID          string `json:"testID"`
	Index       int    `json:"index"`
	ProblemSlug string `json:"problemSlug"`
	Result      string `json:"result"`
	TimeElapsed int    `json:"timeElapsed"`
	MemoryUsed  int    `json:"memoryUsed"`
}

type TestResult struct {
	ID          string `json:"testID"`
	ProblemSlug string `json:"problemSlug"`
	Result      string `json:"result"`
}

type TestRequest struct {
	ProblemSlug string `json:"problemSlug"`
	ID          string `json:"id"`          // test id
	InfoURL     string `json:"infoURL"`     // problem info url
	ProgramURL  string `json:"programURL"`  // user program url
	UserProgram string `json:"userProgram"` // user main program file name
}

type ProblemInfo struct {
	ProblemSlug string `json:"problemSlug"`
	ProblemName string `json:"problemName"`
	Tests       []Test `json:"tests"`
}

type Test struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	InputURL  string `json:"inputURL"`
	OutputURL string `json:"outputURL"`
}

type InnerInputOutputTest struct {
	Inputs  []string `json:"input"`
	Outputs []string `json:"output"`
}

type VmInit struct {
	ProgramType string `json:"programType"`
	UserProgram string `json:"userProgram"`
	IsDirectory bool   `json:"isDirectory"`
	TestCount   int    `json:"testCount"`
}
