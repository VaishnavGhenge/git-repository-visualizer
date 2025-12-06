package http

import (
	"net/http"
	"strconv"
)

func (h *Handler) GetLimitOffset(r *http.Request) (int, int) {
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		limitInt = h.httpCfg.LIMIT
	}

	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		offsetInt = h.httpCfg.OFFSET
	}

	return limitInt, offsetInt
}
	