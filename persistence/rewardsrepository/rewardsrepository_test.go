package rewardsrepository

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/owncast/owncast/core/data"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/migrations"
)

func newTestRepository(t *testing.T) (*SqlRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Run(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO users(id, display_name, display_color, email) VALUES('user-1', 'Kevin', 1, 'kevin@example.com')"); err != nil {
		t.Fatal(err)
	}
	return New(&data.Datastore{DB: db, DbLock: &sync.Mutex{}}).(*SqlRepository), db
}

func enableRewards(t *testing.T, repository Repository) {
	t.Helper()
	settings, err := repository.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	settings.Enabled = true
	settings.SpinCost = 1
	if err := repository.SetSettings(settings); err != nil {
		t.Fatal(err)
	}
}

func TestAwardSpinCreditsCreatesLedgerAndIsIdempotent(t *testing.T) {
	repository, _ := newTestRepository(t)

	balance, err := repository.AwardSpinCredits("user-1", 5, models.RewardCreditSourceTaskCompleted, "task-1", "completed profile task")
	if err != nil {
		t.Fatal(err)
	}
	if balance.Balance != 5 || balance.LifetimeEarned != 5 {
		t.Fatalf("unexpected balance after award: %+v", balance)
	}

	balance, err = repository.AwardSpinCredits("user-1", 5, models.RewardCreditSourceTaskCompleted, "task-1", "duplicate task")
	if err != nil {
		t.Fatal(err)
	}
	if balance.Balance != 5 || balance.LifetimeEarned != 5 {
		t.Fatalf("duplicate reference should not double award: %+v", balance)
	}
}

func TestSpinRequiresCredit(t *testing.T) {
	repository, _ := newTestRepository(t)
	enableRewards(t, repository)

	_, err := repository.SpinWheel(models.User{ID: "user-1", DisplayName: "Kevin"})
	if err == nil {
		t.Fatal("expected spin without credit to fail")
	}
}

func TestSorrySpinDeductsCreditWithoutOrder(t *testing.T) {
	repository, _ := newTestRepository(t)
	enableRewards(t, repository)
	if _, err := repository.AwardSpinCredits("user-1", 1, models.RewardCreditSourceAdminGrant, "grant-1", "seed"); err != nil {
		t.Fatal(err)
	}

	result, err := repository.SpinWheel(models.User{ID: "user-1", DisplayName: "Kevin"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Spin.ResultType != "sorry" || result.Winner != nil || result.Order != nil {
		t.Fatalf("sorry prize should not create fulfilment records: %+v", result)
	}
	balance, err := repository.GetBalance("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if balance.Balance != 0 || balance.LifetimeSpent != 1 {
		t.Fatalf("spin credit deduction incorrect: %+v", balance)
	}
}

func TestRealPrizeSpinCreatesClaimOrderWinnerAndReducesStock(t *testing.T) {
	repository, db := newTestRepository(t)
	enableRewards(t, repository)
	if _, err := db.Exec("UPDATE reward_prizes SET active=0"); err != nil {
		t.Fatal(err)
	}
	stock := 1
	prize, err := repository.UpsertPrize(models.RewardPrize{
		Name:                     "VIP T-shirt",
		PrizeType:                models.RewardPrizeTypePhysical,
		OddsWeight:               100,
		StockQuantity:            &stock,
		Active:                   true,
		ClaimRequired:            true,
		Description:              "Physical prize",
		FulfilmentNotes:          "Pack with sticker",
		Terms:                    "One per viewer",
		MarketingConsentRequired: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.AwardSpinCredits("user-1", 1, models.RewardCreditSourceAdminGrant, "grant-2", "seed"); err != nil {
		t.Fatal(err)
	}

	result, err := repository.SpinWheel(models.User{ID: "user-1", DisplayName: "Kevin", Email: "kevin@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Prize.ID != prize.ID || result.Winner == nil || result.Claim == nil || result.Order == nil {
		t.Fatalf("real prize fulfilment records missing: %+v", result)
	}
	if result.Claim.Status != models.RewardClaimStatusPendingDetails || result.Order.OrderStatus != models.RewardOrderStatusAwaitingClaimDetails {
		t.Fatalf("unexpected claim/order status: claim=%+v order=%+v", result.Claim, result.Order)
	}
	prizes, err := repository.ListPrizes(true)
	if err != nil {
		t.Fatal(err)
	}
	for _, savedPrize := range prizes {
		if savedPrize.ID == prize.ID && (savedPrize.StockQuantity == nil || *savedPrize.StockQuantity != 0) {
			t.Fatalf("expected stock to be reduced to 0: %+v", savedPrize)
		}
	}
}

func TestSubmitClaimMovesOrderReady(t *testing.T) {
	repository, db := newTestRepository(t)
	enableRewards(t, repository)
	if _, err := db.Exec("UPDATE reward_prizes SET active=0"); err != nil {
		t.Fatal(err)
	}
	stock := 2
	if _, err := repository.UpsertPrize(models.RewardPrize{Name: "Slipmat", PrizeType: models.RewardPrizeTypePhysical, OddsWeight: 100, StockQuantity: &stock, Active: true, ClaimRequired: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.AwardSpinCredits("user-1", 1, models.RewardCreditSourceAdminGrant, "grant-3", "seed"); err != nil {
		t.Fatal(err)
	}
	result, err := repository.SpinWheel(models.User{ID: "user-1", DisplayName: "Kevin"})
	if err != nil {
		t.Fatal(err)
	}

	claim, err := repository.SubmitClaim("user-1", models.RewardClaimSubmission{
		ClaimID:      result.Claim.ID,
		FullName:     "Kevin Example",
		AddressLine1: "1 Bass Street",
		TownCity:     "London",
		Postcode:     "SW1A 1AA",
		Country:      "UK",
		Email:        "kevin@example.com",
	}, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if claim.Status != models.RewardClaimStatusSubmitted || claim.MarketingConsent {
		t.Fatalf("unexpected submitted claim: %+v", claim)
	}
	summary, err := repository.GetAdminSummary()
	if err != nil {
		t.Fatal(err)
	}
	if summary.Orders[0].OrderStatus != models.RewardOrderStatusReadyToFulfil {
		t.Fatalf("order was not moved to ready: %+v", summary.Orders[0])
	}
}

func TestDispatchCreatesUserNotification(t *testing.T) {
	repository, db := newTestRepository(t)
	enableRewards(t, repository)
	if _, err := db.Exec("UPDATE reward_prizes SET active=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.UpsertPrize(models.RewardPrize{Name: "Digital download", PrizeType: models.RewardPrizeTypeDigital, OddsWeight: 100, Active: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.AwardSpinCredits("user-1", 1, models.RewardCreditSourceAdminGrant, "grant-4", "seed"); err != nil {
		t.Fatal(err)
	}
	result, err := repository.SpinWheel(models.User{ID: "user-1", DisplayName: "Kevin"})
	if err != nil {
		t.Fatal(err)
	}
	order, notification, err := repository.MarkOrderDispatched("admin-1", result.Order.ID, "Royal Mail", "TRACK123", "https://example.com/track", "Enjoy")
	if err != nil {
		t.Fatal(err)
	}
	if order.OrderStatus != models.RewardOrderStatusDispatched || notification.Type != models.RewardUserNotificationOrderDispatched {
		t.Fatalf("unexpected dispatch result: order=%+v notification=%+v", order, notification)
	}
}

func TestWeightedPrizeSelectionHonorsPositiveWeights(t *testing.T) {
	prize := selectWeightedPrize([]models.RewardPrize{
		{Name: "Never", PrizeType: models.RewardPrizeTypeSorry, OddsWeight: 0},
		{Name: "Always", PrizeType: models.RewardPrizeTypeDigital, OddsWeight: 10},
	})
	if prize.Name != "Always" {
		t.Fatalf("expected positive weighted prize, got %+v", prize)
	}
}
