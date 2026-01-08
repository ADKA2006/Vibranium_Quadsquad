// Package receipts provides PDF receipt generation
package receipts

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/plm/predictive-liquidity-mesh/payments"
)

// getSignatureSecretKey returns the HMAC signing key from environment
// SECURITY: This MUST be set in production via RECEIPT_SIGNATURE_KEY env var
func getSignatureSecretKey() []byte {
	key := os.Getenv("RECEIPT_SIGNATURE_KEY")
	if key == "" {
		log.Println("⚠️  SECURITY WARNING: RECEIPT_SIGNATURE_KEY not set - using insecure default (DEV ONLY)")
		return []byte("plm-dev-receipt-key-NOT-FOR-PRODUCTION")
	}
	return []byte(key)
}

// getUserSalt returns the user ID hashing salt from environment
func getUserSalt() string {
	salt := os.Getenv("USER_ID_SALT")
	if salt == "" {
		log.Println("⚠️  SECURITY WARNING: USER_ID_SALT not set - using insecure default (DEV ONLY)")
		return "plm-dev-salt-NOT-FOR-PRODUCTION"
	}
	return salt
}

// Generator generates PDF receipts for transactions
type Generator struct {
	companyName string
	logoPath    string
}

// NewGenerator creates a new receipt generator
func NewGenerator(companyName string) *Generator {
	return &Generator{
		companyName: companyName,
	}
}

// GeneratePDF generates a PDF receipt for a transaction
func (g *Generator) GeneratePDF(txn *payments.Transaction) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetTextColor(16, 185, 129) // Emerald color
	pdf.CellFormat(190, 15, g.companyName, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 12)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(190, 8, "Transaction Receipt", "", 1, "C", false, 0, "")

	pdf.Ln(10)

	// Status badge
	pdf.SetFont("Helvetica", "B", 14)
	if txn.Status == payments.StatusSuccess {
		pdf.SetTextColor(16, 185, 129)
		pdf.CellFormat(190, 10, "✓ PAYMENT SUCCESSFUL", "", 1, "C", false, 0, "")
	} else if txn.Status == payments.StatusFailed {
		pdf.SetTextColor(239, 68, 68)
		pdf.CellFormat(190, 10, "✗ PAYMENT FAILED", "", 1, "C", false, 0, "")
	} else {
		pdf.SetTextColor(234, 179, 8)
		pdf.CellFormat(190, 10, "⏳ PAYMENT PENDING", "", 1, "C", false, 0, "")
	}

	pdf.Ln(10)

	// Transaction Details Box
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFillColor(248, 250, 252) // Light gray background
	
	startY := pdf.GetY()
	pdf.Rect(10, startY, 190, 45, "F")
	
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(15, startY+5)
	pdf.Cell(40, 8, "Transaction ID:")
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 8, txn.ID)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(15, startY+13)
	pdf.Cell(40, 8, "Date:")
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 8, txn.CreatedAt.Format("January 2, 2006 at 3:04 PM"))

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(15, startY+21)
	pdf.Cell(40, 8, "Payment Method:")
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 8, fmt.Sprintf("Card ending in %s", txn.CardLast4))

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(15, startY+29)
	pdf.Cell(40, 8, "Route:")
	pdf.SetFont("Helvetica", "", 11)
	routeStr := ""
	for i, code := range txn.Route {
		if i > 0 {
			routeStr += " → "
		}
		routeStr += code
	}
	pdf.Cell(0, 8, routeStr)

	pdf.SetXY(15, startY+37)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(40, 8, "Hops:")
	pdf.SetFont("Helvetica", "", 11)
	pdf.Cell(0, 8, fmt.Sprintf("%d countries", len(txn.Route)-1))

	pdf.Ln(55)

	// Amount Section
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, "Payment Summary", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	
	// Table header
	pdf.SetFillColor(229, 231, 235)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(120, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(70, 8, "Amount", "1", 1, "R", true, 0, "")

	// Table rows
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(120, 8, "Original Amount", "1", 0, "L", false, 0, "")
	pdf.CellFormat(70, 8, fmt.Sprintf("$%.2f %s", txn.Amount, txn.Currency), "1", 1, "R", false, 0, "")

	pdf.CellFormat(120, 8, "Platform Fee (1.5%)", "1", 0, "L", false, 0, "")
	pdf.SetTextColor(239, 68, 68)
	pdf.CellFormat(70, 8, fmt.Sprintf("-$%.2f", txn.BaseFee), "1", 1, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.CellFormat(120, 8, fmt.Sprintf("Hop Fees (0.02%% × %d hops)", len(txn.Route)-1), "1", 0, "L", false, 0, "")
	pdf.SetTextColor(239, 68, 68)
	pdf.CellFormat(70, 8, fmt.Sprintf("-$%.2f", txn.HopFees), "1", 1, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	if txn.HaltFines > 0 {
		pdf.CellFormat(120, 8, "Halt Fines (0.1%)", "1", 0, "L", false, 0, "")
		pdf.SetTextColor(239, 68, 68)
		pdf.CellFormat(70, 8, fmt.Sprintf("-$%.2f", txn.HaltFines), "1", 1, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}

	// Total
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetFillColor(16, 185, 129)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(120, 10, "Amount Received", "1", 0, "L", true, 0, "")
	pdf.CellFormat(70, 10, fmt.Sprintf("$%.2f %s", txn.FinalAmount, txn.TargetCurrency), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(10)

	// Hop Details (if available)
	if len(txn.HopResults) > 0 {
		pdf.SetFont("Helvetica", "B", 14)
		pdf.CellFormat(190, 10, "Route Details", "", 1, "L", false, 0, "")

		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(229, 231, 235)
		pdf.CellFormat(30, 7, "From", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, "To", "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, "Status", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, "Latency", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, "Amount In", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, "Amount Out", "1", 1, "C", true, 0, "")

		pdf.SetFont("Helvetica", "", 9)
		for _, hop := range txn.HopResults {
			pdf.CellFormat(30, 7, hop.FromCountry, "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 7, hop.ToCountry, "1", 0, "C", false, 0, "")
			
			if hop.Success {
				pdf.SetTextColor(16, 185, 129)
				pdf.CellFormat(25, 7, "OK", "1", 0, "C", false, 0, "")
			} else {
				pdf.SetTextColor(239, 68, 68)
				pdf.CellFormat(25, 7, "FAILED", "1", 0, "C", false, 0, "")
			}
			pdf.SetTextColor(0, 0, 0)
			
			pdf.CellFormat(30, 7, fmt.Sprintf("%dms", hop.Latency), "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 7, fmt.Sprintf("$%.2f", hop.AmountIn), "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 7, fmt.Sprintf("$%.2f", hop.AmountOut), "1", 1, "C", false, 0, "")
		}
	}

	pdf.Ln(10)

	// Footer
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(190, 6, "This is an automated receipt from Predictive Liquidity Mesh.", "", 1, "C", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Generated on %s", time.Now().Format("January 2, 2006 at 3:04 PM")), "", 1, "C", false, 0, "")

	pdf.Ln(8)

	// Digital Signature Box - Anonymous verification
	signature := generateDigitalSignature(txn)
	verificationCode := generateVerificationCode(txn)
	
	pdf.SetFillColor(30, 41, 59) // Dark slate background
	sigY := pdf.GetY()
	pdf.Rect(10, sigY, 190, 40, "F")
	
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(16, 185, 129) // Emerald
	pdf.SetXY(15, sigY+5)
	pdf.Cell(180, 6, "DIGITAL SIGNATURE - Anonymous Ownership Verification")
	
	pdf.SetFont("Courier", "", 7)
	pdf.SetTextColor(200, 200, 200)
	pdf.SetXY(15, sigY+13)
	pdf.Cell(180, 5, fmt.Sprintf("Signature: %s", signature))
	
	pdf.SetXY(15, sigY+20)
	pdf.Cell(180, 5, fmt.Sprintf("Verification Code: %s", verificationCode))
	
	pdf.SetFont("Helvetica", "I", 7)
	pdf.SetTextColor(150, 150, 150)
	pdf.SetXY(15, sigY+28)
	pdf.MultiCell(180, 4, "This signature proves ownership without revealing user identity. Verify at /verify/receipt", "", "L", false)

	// Output to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// generateDigitalSignature creates an HMAC-SHA256 signature for anonymous verification
// This proves ownership without revealing the user ID to others
func generateDigitalSignature(txn *payments.Transaction) string {
	// Create signature data that includes transaction details but hashes user ID
	data := fmt.Sprintf("%s|%s|%.2f|%s|%s",
		txn.ID,
		hashUserID(txn.UserID), // Anonymous user hash
		txn.Amount,
		txn.Currency,
		txn.CreatedAt.Format(time.RFC3339),
	)
	
	h := hmac.New(sha256.New, getSignatureSecretKey())
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// generateVerificationCode creates a short code for quick verification
func generateVerificationCode(txn *payments.Transaction) string {
	data := fmt.Sprintf("%s|%s", txn.ID, txn.CreatedAt.Format("20060102150405"))
	h := sha256.Sum256([]byte(data))
	// Return first 16 chars for a short verification code
	return fmt.Sprintf("PLM-%s", hex.EncodeToString(h[:])[:16])
}

// hashUserID creates an anonymous hash of the user ID
func hashUserID(userID string) string {
	h := sha256.Sum256([]byte(userID + getUserSalt()))
	return hex.EncodeToString(h[:])[:12] // Short anonymous hash
}

