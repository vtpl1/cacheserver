// Package models contains all the data models
package models

// Recording represents a recording
type Recording struct {
	SiteID       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelID    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
}

// Event represents an event
type Event struct {
	SiteID       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelID    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
}

// Human represents a human
type Human struct {
	SiteID       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelID    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
	// PeopleCount  int    `json:"peopleCount"`
}

// Vehicle represents a vehicle
type Vehicle struct {
	SiteID       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelID    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
	// VehicleCount int    `json:"vehicleCount"`
}

// Result represents the result of a query
type Result struct {
	Recordings []Recording `json:"recording"`
	Events     []Event     `json:"event"`
	Humans     []Human     `json:"human"`
	Vehicles   []Vehicle   `json:"vehicle"`
}

// TimeLineResponse represents the response of a timeline query
type TimeLineResponse struct {
	ReturnValue string   `json:"returnValue"`
	Code        int      `json:"code"`
	Status      int      `json:"status"`
	Description string   `json:"description"`
	Message     string   `json:"message"`
	Results     []Result `json:"result"`
}

// NewTimeLineResponse creates a new TimeLineResponse
func NewTimeLineResponse() TimeLineResponse {
	return TimeLineResponse{
		ReturnValue: "SUCCESS",
		Code:        0,
		Status:      200,
		Description: "OK",
		Message:     "Successfully Retrieved!",
		Results: []Result{
			{Recordings: []Recording{}, Events: []Event{}, Humans: []Human{}, Vehicles: []Vehicle{}},
		},
	}
}

// Command represents a command
type Command struct {
	CommandID  string `json:"commandId"`
	PivotPoint int    `json:"pivotPoint"`
	DisplayMin int    `json:"displayMin"`
	DisplayMax int    `json:"displayMax"`
	DomainMin  int    `json:"domainMin"`
	DomainMax  int    `json:"domainMax"`
}
