package api

type Recording struct {
	SiteId       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelId    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
}

type Event struct {
	SiteId       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelId    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
}

type Human struct {
	SiteId       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelId    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
	PeopleCount  int    `json:"peopleCount"`
}

type Vehicle struct {
	SiteId       int    `json:"siteId,omitempty" bson:"siteId,omitempty"`
	ChannelId    int    `json:"channelId,omitempty" bson:"channelId,omitempty"`
	TimeStamp    uint64 `json:"timeStamp" bson:"startTimestamp"`
	TimeStampEnd uint64 `json:"timeStampEnd" bson:"endTimestamp"`
	VehicleCount int    `json:"vehicleCount"`
}

type Result struct {
	Recordings []Recording `json:"recordings"`
	Events     []Event     `json:"events"`
	Humans     []Human     `json:"humans"`
	Vehicles   []Vehicle   `json:"vehicles"`
}

type TimeLineResponse struct {
	ReturnValue string   `json:"returnValue"`
	Code        int      `json:"code"`
	Status      int      `json:"status"`
	Description string   `json:"description"`
	Message     string   `json:"message"`
	Results     []Result `json:"results"`
}

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
