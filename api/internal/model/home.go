package model

// HomeConfig stores the public blog homepage copy controlled from manage.
type HomeConfig struct {
	Eyebrow         string `json:"eyebrow" orm:"eyebrow"`
	Title           string `json:"title" orm:"title"`
	Subtitle        string `json:"subtitle" orm:"subtitle"`
	SiteTitle       string `json:"siteTitle" orm:"site_title"`
	SiteDescription string `json:"siteDescription" orm:"site_description"`
	SupportEmail    string `json:"supportEmail" orm:"support_email"`
	FooterTagline   string `json:"footerTagline" orm:"footer_tagline"`
	FooterCopyright string `json:"footerCopyright" orm:"footer_copyright"`
}
