package grpc

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	leadflowv1 "leadflow/api/gen/leadflow/v1"
	"leadflow/internal/domain"
	"leadflow/internal/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LeadHandler struct {
	leadflowv1.UnimplementedLeadServiceServer

	leadRepo *repository.LeadRepo
	logger   *slog.Logger
}

func NewLeadHandler(leadRepo *repository.LeadRepo, logger *slog.Logger) *LeadHandler {
	return &LeadHandler{
		leadRepo: leadRepo,
		logger:   logger,
	}
}

func (h *LeadHandler) GetLead(
	ctx context.Context,
	req *leadflowv1.GetLeadRequest,
) (*leadflowv1.GetLeadResponse, error) {
	if req.GetLeadId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "lead_id is required")
	}

	lead, err := h.leadRepo.GetByID(ctx, req.GetLeadId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get lead")
	}

	if lead == nil {
		return nil, status.Error(codes.NotFound, "lead not found")
	}

	return toProtoLead(lead), nil
}

func (h *LeadHandler) ListLeads(
	ctx context.Context,
	req *leadflowv1.ListLeadsRequest,
) (*leadflowv1.ListLeadsResponse, error) {
	pageSize := req.GetPageSize()

	if pageSize == 0 {
		pageSize = 10
	}

	if pageSize > 100 {
		pageSize = 100
	}

	var afterID int64

	if req.GetPageToken() != "" {
		parsed, err := strconv.ParseInt(req.GetPageToken(), 10, 64)
		if err != nil || parsed < 0 {
			return nil, status.Error(
				codes.InvalidArgument,
				"invalid page_token",
			)
		}

		afterID = parsed
	}

	leads, err := h.leadRepo.List(ctx, pageSize, afterID)
	if err != nil {
		h.logger.ErrorContext(
			ctx,
			"failed to list leads",
			"error", err.Error(),
			"page_size", pageSize,
			"after_id", afterID,
		)
		return nil, status.Error(codes.Internal, "failed to list leads")
	}

	resp := &leadflowv1.ListLeadsResponse{
		Leads: make([]*leadflowv1.GetLeadResponse, 0, len(leads)),
	}

	for _, lead := range leads {
		resp.Leads = append(resp.Leads, toProtoLead(lead))
	}

	if len(leads) == int(pageSize) {
		last := leads[len(leads)-1]
		resp.NextPageToken = strconv.FormatInt(last.ID, 10)
	}

	return resp, nil
}

func toProtoLead(lead *domain.Lead) *leadflowv1.GetLeadResponse {
	return &leadflowv1.GetLeadResponse{
		Id:                lead.ID,
		Source:            lead.Source,
		Name:              derefString(lead.Name),
		Email:             derefString(lead.Email),
		Phone:             derefString(lead.Phone),
		Message:           derefString(lead.Message),
		Status:            string(lead.Status),
		BitrixId:          lead.BitrixID,
		TelegramMessageId: lead.TelegramMessageID,
		CreatedAt:         lead.CreatedAt.Format(time.RFC3339),
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
