package handler

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"leadflow/internal/domain"
	"leadflow/internal/service"
)

type AdminLeadsHandler struct {
	svc *service.LeadService
	tpl *template.Template
}

func NewAdminLeadsHandler(
	svc *service.LeadService,
) (*AdminLeadsHandler, error) {
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
	Items        []leadListItem
	Total        int
	StatusFilter string
	SourceFilter string
	Search       string
	Sort         string
	Limit        int
	Offset       int
	HasPrev      bool
	PrevOffset   int
	HasNext      bool
	NextOffset   int
}

func (h *AdminLeadsHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := parsePositiveInt(r.URL.Query().Get("limit"), 50)
	offset := parseInt(r.URL.Query().Get("offset"), 0)

	statusFilter := r.URL.Query().Get("status")
	sourceFilter := r.URL.Query().Get("source")
	search := r.URL.Query().Get("search")
	sort := r.URL.Query().Get("sort")

	var status *domain.LeadStatus

	if statusFilter != "" {
		parsed := domain.LeadStatus(statusFilter)

		if parsed.IsValid() {
			status = &parsed
		}
	}

	var source *string

	if sourceFilter != "" {
		source = &sourceFilter
	}

	leads, total, err := h.svc.ListForOrganization(
		r.Context(),
		service.ListLeadsInput{
			OrganizationID: organizationID,
			Status:         status,
			Source:         source,
			Search:         search,
			Sort:           sort,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	items := make([]leadListItem, 0, len(leads))

	for _, lead := range leads {
		name := derefString(lead.Name)
		email := derefString(lead.Email)
		phone := derefString(lead.Phone)

		items = append(items, leadListItem{
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

	hasNext := offset+len(items) < total
	nextOffset := offset + limit

	data := adminLeadsData{
		Items:        items,
		Total:        total,
		StatusFilter: statusFilter,
		SourceFilter: sourceFilter,
		Search:       search,
		Sort:         sort,
		Limit:        limit,
		Offset:       offset,
		HasPrev:      hasPrev,
		PrevOffset:   prevOffset,
		HasNext:      hasNext,
		NextOffset:   nextOffset,
	}

	w.Header().Set("Content-Type", "text/html, charset=utf-8")

	if err := h.tpl.Execute(w, data); err != nil {
		http.Error(w, "TTemplate error", http.StatusInternalServerError)
	}

}

func parsePositiveInt(
	value string,
	defaultValue int,
) int {
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func parseInt(
	value string,
	defaultValue int,
) int {
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultValue
	}

	return parsed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

type adminCreateLeadRequest struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

func (h *AdminLeadsHandler) CreateLead(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")
	message := r.FormValue("message")
	source := r.FormValue("source")

	if source == "" {
		source = "manual"
	}

	var namePtr *string

	if name != "" {
		namePtr = &name
	}

	var emailPtr *string

	if email != "" {
		emailPtr = &email
	}

	var phonePtr *string

	if phone != "" {
		phonePtr = &phone
	}

	var messagePtr *string
	if message != "" {
		messagePtr = &message
	}

	_, err := h.svc.CreateLead(
		r.Context(),
		service.CreateLeadInput{
			Source:         source,
			Name:           namePtr,
			Email:          emailPtr,
			Phone:          phonePtr,
			Message:        messagePtr,
			OrganizationID: organizationID,
		},
		"",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/admin/leads", http.StatusSeeOther)
}

type adminUpdateStatusRequest struct {
	Status string `json:"status"`
}

func (h *AdminLeadsHandler) UpdateStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := GetOrganizationID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid lead id", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	status := domain.LeadStatus(r.FormValue("status"))

	if !status.IsValid() {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	_, err = h.svc.UpdateStatus(
		r.Context(),
		id,
		organizationID,
		status,
	)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			http.Error(w, "Lead not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Update lead status failed", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/leads", http.StatusSeeOther)
}
