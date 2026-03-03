package files

import (
	"context"
	"encoding/json"
	"log"

	"github.com/mamed-gasimov/file-service/internal/messaging"
)

// ConsumeAnalysisResults reads from "file.analysis.result" and updates the DB.
// It runs until ctx is cancelled. The caller starts it in a goroutine.
func ConsumeAnalysisResults(ctx context.Context, consumer messaging.Consumer, repo repository) {
	msgs, err := consumer.Consume(ctx, "file.analysis.result")
	if err != nil {
		log.Printf("consume analysis results: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case body, ok := <-msgs:
			if !ok {
				return
			}
			var reply AnalysisReply
			if err := json.Unmarshal(body, &reply); err != nil {
				log.Printf("unmarshal analysis reply: %v", err)
				continue
			}
			if reply.Error != "" {
				log.Printf("analysis error for file %d: %s", reply.FileID, reply.Error)
				continue
			}
			if err := repo.UpdateTranslationSummary(ctx, reply.FileID, reply.TranslationSummary); err != nil {
				log.Printf("update translation summary for file %d: %v", reply.FileID, err)
			} else {
				log.Printf("translation summary updated for file %d", reply.FileID)
			}
		}
	}
}
