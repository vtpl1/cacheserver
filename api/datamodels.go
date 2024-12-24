package api

type Recording struct {
	SiteId       int    `json:"siteId"`
	ChannelId    int    `json:"channelId"`
	TimeStamp    uint64 `json:"timeStamp"`
	TimeStampEnd uint64 `json:"timeStampEnd"`
}

type Event struct {
	SiteId       int    `json:"siteId"`
	ChannelId    int    `json:"channelId"`
	TimeStamp    uint64 `json:"timeStamp"`
	TimeStampEnd uint64 `json:"timeStampEnd"`
}

type Human struct {
	SiteId       int    `json:"siteId"`
	ChannelId    int    `json:"channelId"`
	TimeStamp    uint64 `json:"timeStamp"`
	TimeStampEnd uint64 `json:"timeStampEnd"`
	PeopleCount  int    `json:"peopleCount"`
}

type Vehicle struct {
	SiteId       int    `json:"siteId"`
	ChannelId    int    `json:"channelId"`
	TimeStamp    uint64 `json:"timeStamp"`
	TimeStampEnd uint64 `json:"timeStampEnd"`
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
