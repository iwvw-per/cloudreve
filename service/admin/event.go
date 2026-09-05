package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth/requestinfo"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// RecordEvent 写入一条操作/审计日志。IP、UserAgent、correlation_id 从请求上下文自动补全。
func RecordEvent(c *gin.Context, e *inventory.CreateEventParams) {
	dep := dependency.FromContext(c)
	if dep == nil {
		return
	}

	if e.CorrelationID == "" {
		e.CorrelationID = logging.CorrelationID(c).String()
	}
	if ri := requestinfo.RequestInfoFromContext(c); ri != nil {
		if e.IP == "" {
			e.IP = ri.IP
		}
		if e.UserAgent == "" {
			e.UserAgent = ri.UserAgent
		}
	}

	_, _ = dep.EventClient().Create(c, e)
}

type ListAuditLogResponse struct {
	Pagination *inventory.PaginationResults `json:"pagination"`
	Logs       []GetAuditLogResponse        `json:"logs"`
}

type GetAuditLogResponse struct {
	*ent.Event
	UserHashID string `json:"user_hash_id,omitempty"`
}

const (
	eventTypeCondition = "event_type"
	eventUserCondition = "event_user"
	eventIDCondition   = "event_id"
)

func (s *AdminListService) Events(c *gin.Context) (*ListAuditLogResponse, error) {
	dep := dependency.FromContext(c)
	eventClient := dep.EventClient()
	hasher := dep.HashIDEncoder()

	var (
		err      error
		userID   int
		types    []int
		eventIDs []int
	)

	if s.Conditions[eventTypeCondition] != "" {
		for _, tStr := range strings.Split(s.Conditions[eventTypeCondition], ",") {
			t, parseErr := strconv.Atoi(tStr)
			if parseErr != nil {
				return nil, serializer.NewError(serializer.CodeParamErr, "Invalid event type", parseErr)
			}
			types = append(types, t)
		}
	}

	if s.Conditions[eventUserCondition] != "" {
		userID, err = strconv.Atoi(s.Conditions[eventUserCondition])
		if err != nil {
			return nil, serializer.NewError(serializer.CodeParamErr, "Invalid event user ID", err)
		}
	}

	if s.Conditions[eventIDCondition] != "" {
		for _, idStr := range strings.Split(s.Conditions[eventIDCondition], ",") {
			id, parseErr := strconv.Atoi(idStr)
			if parseErr != nil {
				return nil, serializer.NewError(serializer.CodeParamErr, "Invalid event ID", parseErr)
			}
			eventIDs = append(eventIDs, id)
		}
	}

	ctx := context.WithValue(c, inventory.LoadEventUser{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEventFile{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEventEntity{}, true)
	ctx = context.WithValue(ctx, inventory.LoadEventShare{}, true)

	res, err := eventClient.List(ctx, &inventory.ListEventArgs{
		PaginationArgs: &inventory.PaginationArgs{
			Page:     s.Page - 1,
			PageSize: s.PageSize,
			OrderBy:  s.OrderBy,
			Order:    inventory.OrderDirection(s.OrderDirection),
		},
		Types:    types,
		UserID:   userID,
		EventIDs: eventIDs,
	})
	if err != nil {
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to list events", err)
	}

	return &ListAuditLogResponse{
		Pagination: res.PaginationResults,
		Logs: lo.Map(res.Events, func(e *ent.Event, _ int) GetAuditLogResponse {
			var uid string
			if e.Edges.User != nil {
				uid = hashid.EncodeUserID(hasher, e.Edges.User.ID)
			}
			return GetAuditLogResponse{
				Event:      e,
				UserHashID: uid,
			}
		}),
	}, nil
}

type (
	SingleAuditLogService struct {
		ID int
	}
	SingleAuditLogParamCtx struct{}
)

func (s *SingleAuditLogService) Get(c *gin.Context) (*GetAuditLogResponse, error) {
	dep := dependency.FromContext(c)

	event, err := dep.EventClient().GetByID(c, s.ID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, serializer.NewError(serializer.CodeNotFound, "Event not found", err)
		}
		return nil, serializer.NewError(serializer.CodeDBError, "Failed to get event", err)
	}

	var uid string
	if event.Edges.User != nil {
		uid = hashid.EncodeUserID(dep.HashIDEncoder(), event.Edges.User.ID)
	}

	return &GetAuditLogResponse{
		Event:      event,
		UserHashID: uid,
	}, nil
}

type (
	CleanupEventService struct {
		NotAfter time.Time `json:"not_after" binding:"required"`
		Types    []int     `json:"types"`
	}
	CleanupEventParameterCtx struct{}
)

func (s *CleanupEventService) Cleanup(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if _, err := dep.EventClient().Cleanup(c, &inventory.CleanupEventArgs{
		NotAfter: &s.NotAfter,
		Types:    s.Types,
	}); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to cleanup events", err)
	}

	return nil
}

type (
	BatchEventService struct {
		IDs []int `json:"ids" binding:"required"`
	}
	BatchEventParamCtx struct{}
)

func (s *BatchEventService) Delete(c *gin.Context) error {
	dep := dependency.FromContext(c)
	if err := dep.EventClient().Delete(c, s.IDs); err != nil {
		return serializer.NewError(serializer.CodeDBError, "Failed to delete events", err)
	}

	return nil
}