package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVectorSearch(t *testing.T) {
	ctx := context.Background()
	indexName := "my_index"
	schema := map[string]string{
		"field1": "TEXT",
		"field2": "NUMERIC",
	}
	driver := "redis"
	host := []string{"localhost:6379"}
	algorithm := "HNSW"
	dim := 64

	vs := NewVectorSearch(driver, host, ctx, indexName, schema, algorithm, dim)

	err := vs.CreateCollection()
	assert.NoError(t, err)

	key1 := "key1"
	vector1 := []float32{0.1, 0.2, 0.3}
	props1 := map[string]string{
		"field1": "value1",
		"field2": "100",
	}
	err = vs.Add(key1, vector1, props1)
	assert.NoError(t, err)

	key2 := "key2"
	vector2 := []float32{0.2, 0.3, 0.4}
	props2 := map[string]string{
		"field1": "value2",
		"field2": "200",
	}
	err = vs.Add(key2, vector2, props2)
	assert.NoError(t, err)

	// Test search with no tags
	returnFields := []string{"field1", "field2"}
	tags := []string(nil)
	k := 1
	searchVector := []float32{0.15, 0.25, 0.35}
	results, err := vs.Search(k, searchVector, returnFields, tags)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, key1, results[0].Key)
	assert.Equal(t, "0.015", results[0].Score)
	expectedProps := map[string]interface{}{
		"field1": "value1",
		"field2": "100",
	}
	assert.True(t, reflect.DeepEqual(expectedProps, results[0].Props))

	// Test search with tags
	tags = []string{"tag1", "tag2"}
	results, err = vs.Search(k, searchVector, returnFields, tags)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))

	// Test delete
	err = vs.Delete(key1)
	assert.NoError(t, err)
	err = vs.Delete(key2)
	assert.NoError(t, err)
}
