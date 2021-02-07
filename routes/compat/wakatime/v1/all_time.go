package v1

import (
	"github.com/gorilla/mux"
	conf "github.com/muety/wakapi/config"
	"github.com/muety/wakapi/middlewares"
	"github.com/muety/wakapi/models"
	v1 "github.com/muety/wakapi/models/compat/wakatime/v1"
	"github.com/muety/wakapi/services"
	"github.com/muety/wakapi/utils"
	"net/http"
	"net/url"
	"time"
)

type AllTimeHandler struct {
	config      *conf.Config
	userSrvc    services.IUserService
	summarySrvc services.ISummaryService
}

func NewAllTimeHandler(userService services.IUserService, summaryService services.ISummaryService) *AllTimeHandler {
	return &AllTimeHandler{
		userSrvc:    userService,
		summarySrvc: summaryService,
		config:      conf.Get(),
	}
}

func (h *AllTimeHandler) RegisterRoutes(router *mux.Router) {
	r := router.PathPrefix("/compat/wakatime/v1/users/{user}/all_time_since_today").Subrouter()
	r.Use(
		middlewares.NewAuthenticateMiddleware(h.userSrvc).Handler,
	)
	r.Methods(http.MethodGet).HandlerFunc(h.Get)
}

// @Summary Retrieve summary for all time
// @Description Mimics https://wakatime.com/developers#all_time_since_today
// @ID get-all-time
// @Tags wakatime
// @Produce json
// @Param user path string true "User ID to fetch data for (or 'current')"
// @Security ApiKeyAuth
// @Success 200 {object} v1.AllTimeViewModel
// @Router /compat/wakatime/v1/users/{user}/all_time_since_today [get]
func (h *AllTimeHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	values, _ := url.ParseQuery(r.URL.RawQuery)

	requestedUser := vars["user"]
	authorizedUser := r.Context().Value(models.UserKey).(*models.User)

	if requestedUser != authorizedUser.ID && requestedUser != "current" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	summary, err, status := h.loadUserSummary(authorizedUser)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}

	vm := v1.NewAllTimeFrom(summary, models.NewFiltersWith(models.SummaryProject, values.Get("project")))
	utils.RespondJSON(w, http.StatusOK, vm)
}

func (h *AllTimeHandler) loadUserSummary(user *models.User) (*models.Summary, error, int) {
	summaryParams := &models.SummaryParams{
		From:      time.Time{},
		To:        time.Now(),
		User:      user,
		Recompute: false,
	}

	var retrieveSummary services.SummaryRetriever = h.summarySrvc.Retrieve
	if summaryParams.Recompute {
		retrieveSummary = h.summarySrvc.Summarize
	}

	summary, err := h.summarySrvc.Aliased(summaryParams.From, summaryParams.To, summaryParams.User, retrieveSummary)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	return summary, nil, http.StatusOK
}
