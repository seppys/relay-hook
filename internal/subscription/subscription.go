package subscription

import (
	"relay-hook/internal/event"

	"github.com/google/uuid"
)

type Subscriber struct {
	Id          string `json:"id"`
	UserID      string `json:"user_id"`
	EndpointURL string `json:"endpoint_url"`
}

type Subscription struct {
	Id           string     `json:"id"`
	SubscriberId string     `json:"subscriber_id"`
	EventType    event.Type `json:"event_type"`
}

type View struct {
	Id           string     `json:"id"`
	SubscriberId string     `json:"subscriber_id"`
	UserId       string     `json:"user_id"`
	EventType    event.Type `json:"event_type"`
	EndpointURL  string     `json:"endpoint_url"`
}

func NewSubscriber(userID, endpoint string) Subscriber {
	return Subscriber{
		Id:          uuid.New().String(),
		UserID:      userID,
		EndpointURL: endpoint,
	}
}

func NewSubscription(subscriberId string, eventType event.Type) Subscription {
	return Subscription{
		Id:           uuid.New().String(),
		SubscriberId: subscriberId,
		EventType:    eventType,
	}
}
