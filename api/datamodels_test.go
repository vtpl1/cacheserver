package api_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vtpl1/cacheserver/api"
)

func TestTimeLineResponseAppend(t *testing.T) {
	timeLineResponse1 := api.TimeLineResponse{
		ReturnValue: "SUCCESS",
		Code:        0,
		Status:      200,
		Description: "OK",
		Message:     "Successfully Retrieved!",
		Results: []api.Result{
			{
				Recordings: []api.Recording{
					{
						SiteID:       5,
						ChannelID:    5,
						TimeStamp:    1735050709524,
						TimeStampEnd: 1735051309524,
					},
				},
				Events:   []api.Event{},
				Humans:   []api.Human{},
				Vehicles: []api.Vehicle{},
			},
		},
	}
	timeLineResponse2 := api.NewTimeLineResponse()

	recordings := []api.Recording{
		{
			SiteID:       5,
			ChannelID:    5,
			TimeStamp:    1735050709524,
			TimeStampEnd: 1735051309524,
		},
	}

	timeLineResponse2.Results[0].Recordings = recordings
	assert.Equal(t, timeLineResponse1, timeLineResponse2, "Instances must be same")
}

func TestTimeLineResponseDeserialize(t *testing.T) {
	// Open the JSON file
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	fileName := filepath.Join(dir, "..", "response.json")
	fileName = filepath.Clean(fileName)
	jsonFile, err := os.Open(fileName)
	if err != nil {
		t.Fatalf("Failed to open JSON file: %s", err)
	}
	defer jsonFile.Close() //nolint:errcheck

	// Read the file contents
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %s", err)
	}

	// Unmarshal the JSON data into the struct
	var timeLineResponse api.TimeLineResponse
	if err = json.Unmarshal(byteValue, &timeLineResponse); err != nil {
		t.Fatalf("Failed to unmarshal JSON data: %s", err)
	}

	// Print the struct to verify the data
	b, err := json.Marshal(timeLineResponse)
	if err != nil {
		t.Fatalf("Failed to marshal JSON data: %s", err)
	}
	timeLineResponse1 := api.TimeLineResponse{
		ReturnValue: "SUCCESS",
		Code:        0,
		Status:      200,
		Description: "OK",
		Message:     "Successfully Retrieved!",
		Results: []api.Result{
			{
				Recordings: []api.Recording{
					{
						SiteID:       5,
						ChannelID:    5,
						TimeStamp:    1735050709524,
						TimeStampEnd: 1735051309524,
					},
				},
				Events:   []api.Event{},
				Humans:   []api.Human{},
				Vehicles: []api.Vehicle{},
			},
		},
	}

	assert.Equal(t, timeLineResponse1, timeLineResponse, "Elements should be same")
	t.Logf("Name: %v %s\n", timeLineResponse, string(b))
}
