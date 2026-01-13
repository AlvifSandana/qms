package store

import "qms/queue-service/internal/models"

var transitionMap = map[string][]string{
	"call_next":     {models.StatusWaiting},
	"start_serving": {models.StatusCalled},
	"complete":      {models.StatusServing},
	"cancel":        {models.StatusWaiting},
	"hold":          {models.StatusWaiting},
	"unhold":        {models.StatusHeld},
	"recall":        {models.StatusCalled},
	"transfer":      {models.StatusWaiting, models.StatusCalled, models.StatusServing},
	"no_show":       {models.StatusCalled},
}

func ValidTransition(action, fromStatus string) bool {
	allowed, ok := transitionMap[action]
	if !ok {
		return false
	}
	for _, status := range allowed {
		if status == fromStatus {
			return true
		}
	}
	return false
}
