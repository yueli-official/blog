package model

// HomeConfig stores the public blog homepage copy controlled from manage.
type HomeConfig struct {
	Eyebrow  string `json:"eyebrow" orm:"eyebrow"`
	Title    string `json:"title" orm:"title"`
	Subtitle string `json:"subtitle" orm:"subtitle"`
}
