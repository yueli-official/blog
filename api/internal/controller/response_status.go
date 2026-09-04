package controller

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"
)

func writeCreated(ctx context.Context) {
	ghttp.RequestFromCtx(ctx).Response.WriteHeader(http.StatusCreated)
}

func writeNoContent(ctx context.Context) {
	ghttp.RequestFromCtx(ctx).Response.WriteHeader(http.StatusNoContent)
}
