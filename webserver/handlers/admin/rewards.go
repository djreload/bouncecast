package admin

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/owncast/owncast/core/rewards"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/rewardsrepository"
	webutils "github.com/owncast/owncast/webserver/utils"
)

func GetRewardsAdmin(w http.ResponseWriter, r *http.Request) {
	summary, err := rewardsrepository.Get().GetAdminSummary()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, summary)
}

func SetRewardsSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var settings models.RewardSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid settings")
		return
	}
	if err := rewardsrepository.Get().SetSettings(settings); err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	webutils.WriteSimpleResponse(w, true, "Rewards settings saved")
}

func UpsertRewardPrize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var prize models.RewardPrize
	if err := json.NewDecoder(r.Body).Decode(&prize); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid prize")
		return
	}
	saved, err := rewardsrepository.Get().UpsertPrize(prize)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, saved)
}

func AdjustRewardCredits(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var request struct {
		UserID string `json:"userId"`
		Amount int    `json:"amount"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.UserID == "" || request.Amount == 0 {
		webutils.WriteSimpleResponse(w, false, "invalid credit adjustment")
		return
	}
	balance, err := rewardsrepository.Get().AdminAdjustSpinCredits(request.UserID, request.Amount, request.Note)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, balance)
}

func AwardTopSupporterRewards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	results, err := rewards.GetService().AwardTopSupporterRewards()
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, webutils.J{"results": results})
}

func UpdateRewardOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var order models.RewardOrder
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid order")
		return
	}
	saved, err := rewardsrepository.Get().UpdateOrder("admin", order)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, saved)
}

func DispatchRewardOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var request struct {
		OrderID           int64  `json:"orderId"`
		Courier           string `json:"courier"`
		TrackingReference string `json:"trackingReference"`
		TrackingURL       string `json:"trackingUrl"`
		DispatchNote      string `json:"dispatchNote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.OrderID <= 0 {
		webutils.WriteSimpleResponse(w, false, "invalid dispatch request")
		return
	}
	order, notification, err := rewards.GetService().DispatchOrder("admin", request.OrderID, request.Courier, request.TrackingReference, request.TrackingURL, request.DispatchNote)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, webutils.J{"order": order, "notification": notification})
}

func MarkRewardAdminMessageRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var request struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid message")
		return
	}
	if err := rewardsrepository.Get().MarkAdminMessageRead(request.ID); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteSimpleResponse(w, true, "Message marked read")
}

func UpsertRewardTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var task models.RewardTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid task")
		return
	}
	saved, err := rewardsrepository.Get().UpsertTask(task)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, saved)
}

func UpsertRewardAchievement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var achievement models.RewardAchievement
	if err := json.NewDecoder(r.Body).Decode(&achievement); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid achievement")
		return
	}
	saved, err := rewardsrepository.Get().UpsertAchievement(achievement)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, saved)
}

func ExportRewardOrdersCSV(w http.ResponseWriter, r *http.Request) {
	summary, err := rewardsrepository.Get().GetAdminSummary()
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=bouncecast-reward-orders.csv")
	writer := csv.NewWriter(w)
	defer writer.Flush()
	_ = writer.Write([]string{"order_id", "user_id", "username", "prize", "status", "dispatch_status", "courier", "tracking", "created_at"})
	for _, order := range summary.Orders {
		_ = writer.Write([]string{
			strconv.FormatInt(order.ID, 10),
			order.WinnerUserID,
			order.UsernameSnapshot,
			order.PrizeSnapshot,
			order.OrderStatus,
			order.DispatchStatus,
			order.Courier,
			fmt.Sprintf("%s %s", order.TrackingReference, order.TrackingURL),
			order.CreatedAt.Format(timeFormatRFC3339),
		})
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"
