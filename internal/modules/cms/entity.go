package cms

type Content struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type InputDTO struct {
	Key   string
	Value string
}
