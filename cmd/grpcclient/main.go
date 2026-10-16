package main

import (
	"context"
	"log"
	"os"
	"time"

	leadflowv1 "leadflow/api/gen/leadflow/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	conn, err := grpc.NewClient(
		"localhost:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to create grpc client: %v", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	client := leadflowv1.NewLeadServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	internalToken := os.Getenv("GRPC_INTERNAL_TOKEN")
	if internalToken == "" {
		log.Fatal("GRPC_INTERNAL_TOKEN is not set")
	}

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"authorization",
		"Bearer "+internalToken,
	)

	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"x-request-id",
		"grpc-manual-test",
	)

	getResp, err := client.GetLead(ctx, &leadflowv1.GetLeadRequest{
		LeadId: 44,
	})
	if err != nil {
		log.Fatalf("GetLead failed: %v", err)
	}

	log.Printf(
		"GetLead: id=%d source=%s status=%s bitrix_id=%d telegram_message_id=%d",
		getResp.GetId(),
		getResp.GetSource(),
		getResp.GetStatus(),
		getResp.GetBitrixId(),
		getResp.GetTelegramMessageId(),
	)

	listResp, err := client.ListLeads(ctx, &leadflowv1.ListLeadsRequest{
		PageSize: 2,
	})
	if err != nil {
		log.Fatalf("ListLeads failed: %v", err)
	}

	if listResp.GetNextPageToken() != "" {
		secondPage, err := client.ListLeads(ctx, &leadflowv1.ListLeadsRequest{
			PageSize:  2,
			PageToken: listResp.GetNextPageToken(),
		})
		if err != nil {
			log.Fatalf("ListLeads page 2 failed: %v", err)
		}
		for _, lead := range secondPage.GetLeads() {
			log.Printf(
				"ListLeads page 2: id=%d source=%s status=%s",
				lead.GetId(),
				lead.GetSource(),
				lead.GetStatus(),
			)
		}

		log.Printf(
			"ListLeads page 2: next_page_token=%q",
			secondPage.GetNextPageToken(),
		)
	}
	for _, lead := range listResp.GetLeads() {
		log.Printf(
			"ListLeads: id=%d source=%s status=%s",
			lead.GetId(),
			lead.GetSource(),
			lead.GetStatus(),
		)
	}

	log.Printf(
		"ListLeads: next_page_token=%q",
		listResp.GetNextPageToken(),
	)
}
