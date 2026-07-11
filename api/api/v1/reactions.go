package v1

import "github.com/gogf/gf/v2/frame/g"

// LikeReq toggles the caller's like on a post (author JWT).
type LikeReq struct {
	g.Meta `path:"/api/v1/posts/{slug}/like" method:"post" tags:"blog" summary:"Toggle like on a post"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type LikeRes struct {
	Liked bool `json:"liked"`
}

// BookmarkReq toggles the caller's bookmark on a post (author JWT).
type BookmarkReq struct {
	g.Meta `path:"/api/v1/posts/{slug}/bookmark" method:"post" tags:"blog" summary:"Toggle bookmark on a post"`
	Slug   string `json:"slug" in:"path" v:"required"`
}

type BookmarkRes struct {
	Bookmarked bool `json:"bookmarked"`
}
