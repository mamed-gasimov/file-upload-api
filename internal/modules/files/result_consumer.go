package files

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/mamed-gasimov/file-service/internal/messaging"
)

// ConsumeAnalysisResults reads from "file.analysis.result" and updates the DB.
// It runs until ctx is cancelled. The caller starts it in a goroutine.
// It reconnects automatically if the message channel closes.
func ConsumeAnalysisResults(ctx context.Context, consumer messaging.Consumer, repo repository) {
	for {
		if ctx.Err() != nil {
			return
		}

		msgs, err := consumer.Consume(ctx, "file.analysis.result")
		if err != nil {
			log.Printf("consume analysis results: %v — retrying in 5s", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		log.Println("result consumer: listening on file.analysis.result")
		for d := range msgs {
			if ctx.Err() != nil {
				return
			}
			if err := handleAnalysisReply(ctx, d, repo); err != nil {
				log.Printf("handle analysis reply: %v, sending to DLQ", err)
				_ = d.Nack(false)
			} else {
				_ = d.Ack()
			}
		}
		log.Println("result consumer: message channel closed, reconnecting…")
	}
}

func handleAnalysisReply(ctx context.Context, d messaging.Delivery, repo repository) error {
	var reply AnalysisReply
	if err := json.Unmarshal(d.Body, &reply); err != nil {
		return err
	}

	if reply.Error != "" {
		log.Printf("analysis error for file %d: %s", reply.FileID, reply.Error)
		return nil
	}

	if err := repo.UpdateTranslationSummary(ctx, reply.FileID, reply.TranslationSummary); err != nil {
		return err
	}

	log.Printf("translation summary updated for file %d", reply.FileID)
	return nil
}
