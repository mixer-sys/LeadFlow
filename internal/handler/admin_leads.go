package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"leadflow/internal/service"
)

type AdminLeadsHandler struct {
	svc *service.LeadService
	tpl *template.Template
}

func NewAdminLeadsHandler(svc *service.LeadService) (*AdminLeadsHandler, error) {
	tpl, err := template.ParseFiles("templates/admin/leads.html")
	if err != nil {
		return nil, err
	}

	return &AdminLeadsHandler{
		svc: svc,
		tpl: tpl,
	}, nil
}

type adminLeadsData struct {
	Items      []*leadListItem
	Total      int64
	HasPrev    bool
	PrevOffset int
	HasNext    bool
	NextOffset int
	Limit      int
}

func (h *AdminLeadsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	result, err := h.svc.ListLeads(r.Context(), service.ListLeadsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	items := make([]*leadListItem, 0, len(result.Items))

	for _, lead := range result.Items {
		name := ""
		if lead.Name != nil {
			name = *lead.Name
		}

		email := ""
		if lead.Email != nil {
			email = *lead.Email
		}

		phone := ""
		if lead.Phone != nil {
			phone = *lead.Phone
		}

		items = append(items, &leadListItem{
			ID:        lead.ID,
			Source:    lead.Source,
			Name:      name,
			Email:     email,
			Phone:     phone,
			Status:    string(lead.Status),
			CreatedAt: lead.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	hasPrev := offset > 0
	prevOffset := 0
	if hasPrev {
		prevOffset = offset - limit
		if prevOffset < 0 {
			prevOffset = 0
		}
	}

	hasNext := int64(offset+len(items)) < result.Total
	nextOffset := offset + limit

	data := adminLeadsData{
		Items:      items,
		Total:      result.Total,
		HasPrev:    hasPrev,
		PrevOffset: prevOffset,
		HasNext:    hasNext,
		NextOffset: nextOffset,
		Limit:      limit,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = h.tpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
