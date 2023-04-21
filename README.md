# go-vectorsearch
Vector Search Interface (supported: Redisearch)

Example usage:
```go

package main

import (
	"context"
	"fmt"
    "context"
	
    "github.com/mragungsetiaji/go-vectorsearch"
)

func main(){
    ctx := context.Background()
	vs := NewVectorSearch({
        Context: ctx,
        Collection: "my_index",
        Schema: NewSchema({
            "field1": "TEXT",
            "field2": "NUMERIC"
        }),
        Driver: "redis",
        Host: []string{"localhost:6379"},
        Algorithm: "HNSW",
        Dim: 3,
    })

	err := vs.CreateCollection()

    // Add document
    key1 := "key1"
	vector1 := []float32{0.1, 0.2, 0.3}
	props1 := map[string]string{
		"field1": "value1",
		"field2": "100",
	}
	err = vs.Add(key1, vector1, props1)

    // Search
    returnFields := []string{"field1", "field2"}
	tags := []string(nil)
	k := 1
	searchVector := []float32{0.15, 0.25, 0.35}
	results, err := vs.Search(k, searchVector, returnFields, tags)

    // Delete document
    err = vs.Delete(key1)
}
```

