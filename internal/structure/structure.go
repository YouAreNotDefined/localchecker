package structure

type Mime struct {
	contentType string
	category    string
}

type Res struct {
	Error    error
	Response string
}
