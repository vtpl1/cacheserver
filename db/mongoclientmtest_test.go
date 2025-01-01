package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vtpl1/cacheserver/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestMockMongoClient(t *testing.T) {
	fileDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fileName := filepath.Join(fileDir, "..", "testdatasuite", "pvaDB.pva_HUMAN_1_1.json")
	dd, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}

	var doc []interface{}
	err = bson.UnmarshalExtJSON(dd, true, &doc)
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := mongo.NewCursorFromDocuments(doc, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()
	count := 0
	for cursor.Next(ctx) {
		var result models.Recording
		if err = cursor.Decode(&result); err != nil {
			break
		}
		count++
	}
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, count, 73)
}
