package starsrepository

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/migrations"
)

func newTestRepository(t *testing.T) Repository {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Run(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id, display_name, display_color) VALUES('user-1', 'Kevin', 1)"); err != nil {
		t.Fatal(err)
	}
	return New(&data.Datastore{DB: db, DbLock: &sync.Mutex{}})
}

func TestPackageValidation(t *testing.T) {
	repository := newTestRepository(t)
	_, err := repository.UpsertPackage(models.StarPackage{Name: "", StarAmount: 100, PriceCents: 100, Currency: "GBP", Enabled: true})
	if err == nil {
		t.Fatal("expected invalid package to fail")
	}

	pkg, err := repository.UpsertPackage(models.StarPackage{Name: "Test Stars", StarAmount: 100, PriceCents: 100, Currency: "GBP", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if pkg.ID == 0 || pkg.Currency != "GBP" {
		t.Fatalf("unexpected package: %+v", pkg)
	}
}

func TestCompletedPayPalOrderCreditsOnce(t *testing.T) {
	repository := newTestRepository(t)
	pkg, err := repository.GetPackage(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreatePayPalOrder("user-1", pkg, "ORDER-1"); err != nil {
		t.Fatal(err)
	}
	if _, credited, err := repository.CompletePayPalOrder("ORDER-1", "CAPTURE-1", "PAYER-1", "COMPLETED"); err != nil || !credited {
		t.Fatalf("first completion failed credited=%v err=%v", credited, err)
	}
	if _, credited, err := repository.CompletePayPalOrder("ORDER-1", "CAPTURE-1", "PAYER-1", "COMPLETED"); err != nil || credited {
		t.Fatalf("duplicate completion should not credit again credited=%v err=%v", credited, err)
	}

	summary, err := repository.GetWalletSummary("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Wallet.Balance != pkg.StarAmount || summary.Wallet.LifetimePurchased != pkg.StarAmount {
		t.Fatalf("wallet credited incorrectly: %+v package=%+v", summary.Wallet, pkg)
	}
}

func TestSendingStarsDeductsAndPreventsNegativeBalance(t *testing.T) {
	repository := newTestRepository(t)
	pkg, err := repository.GetPackage(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreatePayPalOrder("user-1", pkg, "ORDER-2"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = repository.CompletePayPalOrder("ORDER-2", "CAPTURE-2", "PAYER-1", "COMPLETED"); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.SendStars("user-1", "Kevin", 25, "Big up", "sparkle"); err != nil {
		t.Fatal(err)
	}
	if _, err = repository.SendStars("user-1", "Kevin", pkg.StarAmount, "Too much", "sparkle"); err == nil {
		t.Fatal("expected insufficient balance to fail")
	}

	summary, err := repository.GetWalletSummary("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Wallet.Balance != pkg.StarAmount-25 || summary.Wallet.LifetimeSent != 25 {
		t.Fatalf("wallet send state incorrect: %+v", summary.Wallet)
	}
}

func TestRefundCreatesLedgerEntry(t *testing.T) {
	repository := newTestRepository(t)
	pkg, err := repository.GetPackage(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repository.CreatePayPalOrder("user-1", pkg, "ORDER-3"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = repository.CompletePayPalOrder("ORDER-3", "CAPTURE-3", "PAYER-1", "COMPLETED"); err != nil {
		t.Fatal(err)
	}
	if err = repository.RefundPayPalCapture("CAPTURE-3", "REFUNDED"); err != nil {
		t.Fatal(err)
	}

	summary, err := repository.GetWalletSummary("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Wallet.Balance != 0 {
		t.Fatalf("refund should remove unspent stars, got balance %d", summary.Wallet.Balance)
	}
	foundRefund := false
	for _, transaction := range summary.Transactions {
		if transaction.TransactionType == models.StarTransactionPayPalRefund {
			foundRefund = true
		}
	}
	if !foundRefund {
		t.Fatal("expected refund ledger entry")
	}
}
