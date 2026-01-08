// Package handlers provides receipt download endpoints
package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/plm/predictive-liquidity-mesh/payments"
	"github.com/plm/predictive-liquidity-mesh/receipts"
)

// ReceiptHandler handles receipt download requests
type ReceiptHandler struct {
	txnStore  *payments.TransactionStore
	generator *receipts.Generator
}

// NewReceiptHandler creates a new receipt handler
func NewReceiptHandler(txnStore *payments.TransactionStore) *ReceiptHandler {
	return &ReceiptHandler{
		txnStore:  txnStore,
		generator: receipts.NewGenerator("Predictive Liquidity Mesh"),
	}
}

// HandleDownloadReceipt generates and downloads a PDF receipt
func (h *ReceiptHandler) HandleDownloadReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get transaction ID from query or path
	txnID := r.URL.Query().Get("id")
	if txnID == "" {
		// Try to get from path: /api/v1/receipts/{txnID}
		path := r.URL.Path
		if len(path) > len("/api/v1/receipts/") {
			txnID = path[len("/api/v1/receipts/"):]
		}
	}

	if txnID == "" {
		http.Error(w, `{"error":"transaction id required"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📄 Generating receipt for transaction: %s", txnID)

	// Get transaction
	txn, err := h.txnStore.GetTransaction(txnID)
	if err != nil {
		log.Printf("❌ Receipt error: transaction not found: %s", txnID)
		http.Error(w, `{"error":"transaction not found"}`, http.StatusNotFound)
		return
	}

	// Generate PDF
	pdfBytes, err := h.generator.GeneratePDF(txn)
	if err != nil {
		log.Printf("❌ Receipt PDF generation error: %v", err)
		http.Error(w, `{"error":"failed to generate receipt: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Receipt generated: %d bytes for txn %s", len(pdfBytes), txnID)

	// Set headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=receipt_%s.pdf", txnID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Write(pdfBytes)
}

