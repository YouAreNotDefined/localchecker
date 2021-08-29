package structure

type Mime struct {
	contentType string
	category    string
}

type Res struct {
	Error    error
	Response string
}

type Necessary struct {
	include   bool
	includeId bool
}
