// Package blogclient is the blog site's gateway to the asset service for cover
// images. Covers are public assets (unsigned delivery), so this is a slim
// upload/finalize/delete client — no signed-GET / service-token path.
package blogclient

import "context"

// InitInput begins a cover upload on the asset service.
type InitInput struct {
	Filename     string
	Mime         string
	Category     string
	Visibility   string // public
	Size         int64
	Target       string
	Preprocessed bool
}

// InitOutput is the asset service's short link the client PUTs the blob to.
type InitOutput struct {
	UploadURL     string
	UploadToken   string
	UploadHeaders map[string]string
}

// View is the finalized asset metadata the blog snapshots onto the post.
type View struct {
	ID       string
	MediaKey string
	Size     int64
	Mime     string
	Filename string
}

type ReferenceInput struct {
	AssetID, RefType, RefID string
	RefLabel, RefURL        string
}

// Client is the asset-service contract the blog site depends on (covers only).
type Client interface {
	// UploadInit validates + reserves an upload, returning the blob short link.
	UploadInit(ctx context.Context, bearer string, in InitInput) (InitOutput, error)
	// Finalize turns an uploaded blob into a recorded asset and returns its view.
	Finalize(ctx context.Context, bearer, uploadToken string) (View, error)
	RegisterReference(ctx context.Context, bearer string, in ReferenceInput) error
	UnregisterReference(ctx context.Context, bearer string, in ReferenceInput) error
	// Delete removes an asset (best-effort cleanup of a replaced cover).
	Delete(ctx context.Context, bearer, assetID string) error
}
