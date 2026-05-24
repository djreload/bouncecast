package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/owncast/owncast/core/rewards"
	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/persistence/rewardsrepository"
	"github.com/owncast/owncast/utils"
	"github.com/owncast/owncast/webserver/router/middleware"
	webutils "github.com/owncast/owncast/webserver/utils"
)

func GetRewardsWheel(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	data, err := rewards.GetService().GetWheelData(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, data)
}

func GetRewardsBalance(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	balance, err := rewardsrepository.Get().GetBalance(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, balance)
}

func SpinRewardsWheel(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastRewardsSpinRateLimit, bounceCastRateLimitUserSubject(user.ID), bounceCastRateLimitIPSubject(r)) {
		return
	}

	result, err := rewards.GetService().SpinWheel(user)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, result)
}

func GetRewardsHistory(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	spins, err := rewardsrepository.Get().ListUserSpins(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, spins)
}

func CompleteRewardTask(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastRewardsTaskRateLimit, bounceCastRateLimitUserSubject(user.ID), bounceCastRateLimitIPSubject(r)) {
		return
	}
	var request struct {
		TaskID int64 `json:"taskId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.TaskID <= 0 {
		webutils.WriteSimpleResponse(w, false, "task ID is required")
		return
	}
	completion, balance, awarded, err := rewards.GetService().CompleteTask(user.ID, request.TaskID)
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, webutils.J{"completion": completion, "balance": balance, "awarded": awarded})
}

func GetRewardsClaims(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	claims, err := rewardsrepository.Get().ListUserClaims(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, claims)
}

func SubmitRewardsClaim(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	if !enforceBounceCastRateLimit(w, r, bounceCastRewardsClaimRateLimit, bounceCastRateLimitUserSubject(user.ID), bounceCastRateLimitIPSubject(r)) {
		return
	}

	var submission models.RewardClaimSubmission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		webutils.WriteSimpleResponse(w, false, "invalid claim")
		return
	}
	claim, err := rewardsrepository.Get().SubmitClaim(user.ID, submission, utils.GetIPAddressFromRequest(r), r.UserAgent())
	if err != nil {
		webutils.WriteSimpleResponse(w, false, err.Error())
		return
	}
	writeJSON(w, claim)
}

func GetRewardsNotifications(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	notifications, err := rewardsrepository.Get().ListUserNotifications(user.ID)
	if err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	writeJSON(w, notifications)
}

func MarkRewardsNotificationRead(user models.User, w http.ResponseWriter, r *http.Request) {
	middleware.EnableCors(w)
	if r.Method != http.MethodPost {
		webutils.WriteSimpleResponse(w, false, r.Method+" not supported")
		return
	}
	var request struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		if id, parseErr := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64); parseErr == nil {
			request.ID = id
		}
	}
	if request.ID <= 0 {
		webutils.WriteSimpleResponse(w, false, "notification ID is required")
		return
	}
	if err := rewardsrepository.Get().MarkUserNotificationRead(user.ID, request.ID); err != nil {
		webutils.InternalErrorHandler(w, err)
		return
	}
	webutils.WriteSimpleResponse(w, true, "Notification marked read")
}
