package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/cto-up/cron-lib/api/openapi"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// gin serialises a bare error value as `{}`, so an admin hitting a failure got
// a 500 with no hint of what failed. Every error body must carry its message.
func TestHandlersExplainMissingTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &CronHandler{
		JobAuditLogHandler:   newJobAuditLogHandler(nil, nil),
		RegisteredJobHandler: newRegisteredJobHandler(nil, nil),
	}
	id := uuid.New()

	cases := map[string]func(c *gin.Context){
		"DeleteJob":           func(c *gin.Context) { h.DeleteJob(c, id) },
		"GetJobByID":          func(c *gin.Context) { h.GetJobByID(c, id, api.GetJobByIDParams{}) },
		"ListJobs":            func(c *gin.Context) { h.ListJobs(c, api.ListJobsParams{}) },
		"DeleteJobAuditLog":   func(c *gin.Context) { h.DeleteJobAuditLog(c, id) },
		"GetJobAuditLogByID":  func(c *gin.Context) { h.GetJobAuditLogByID(c, id) },
		"ListJobAuditLogs":    func(c *gin.Context) { h.ListJobAuditLogs(c, api.ListJobAuditLogsParams{}) },
		"ListRegisteredJobs":  func(c *gin.Context) { h.ListRegisteredJobs(c, api.ListRegisteredJobsParams{}) },
		"GetRegisteredJob":    func(c *gin.Context) { h.GetRegisteredJob(c, id) },
		"UpdateRegisteredJob": func(c *gin.Context) { h.UpdateRegisteredJob(c, id) },
		"GetJobAuditLogs":     func(c *gin.Context) { h.GetJobAuditLogs(c, id, api.GetJobAuditLogsParams{}) },
	}

	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			call(c)

			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", w.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not JSON: %q", w.Body.String())
			}
			msg, _ := body["error"].(string)
			if msg == "" {
				msg, _ = body["message"].(string)
			}
			if msg == "" {
				t.Fatalf("error body carries no message: %s", w.Body.String())
			}
		})
	}
}
