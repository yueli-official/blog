package model

// FriendLink is an ordered external site reference displayed in the public
// footer. The order in HomeConfig is the display order.
type FriendLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// ContactLink is one ordered public contact method. URL is optional so
// non-linkable identifiers such as a QQ group number remain useful.
type ContactLink struct {
	Value string `json:"value"`
	URL   string `json:"url"`
}

// HomeConfig stores the public blog homepage copy controlled from manage.
type HomeConfig struct {
	Eyebrow          string        `json:"eyebrow" orm:"eyebrow"`
	Title            string        `json:"title" orm:"title"`
	Subtitle         string        `json:"subtitle" orm:"subtitle"`
	SiteTitle        string        `json:"siteTitle" orm:"site_title"`
	SiteDescription  string        `json:"siteDescription" orm:"site_description"`
	SupportEmail     string        `json:"supportEmail" orm:"support_email"`
	FooterTagline    string        `json:"footerTagline" orm:"footer_tagline"`
	FooterCopyright  string        `json:"footerCopyright" orm:"footer_copyright"`
	FriendLinks      []FriendLink  `json:"friendLinks" orm:"-"`
	FriendLinksJSON  string        `json:"-" orm:"friend_links"`
	ContactLinks     []ContactLink `json:"contactLinks" orm:"-"`
	ContactLinksJSON string        `json:"-" orm:"contact_links"`
}
