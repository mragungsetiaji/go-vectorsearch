package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/rueian/rueidis"
)

const DIM = 768

type VectorSearch struct {
	client    rueidis.Client
	ctx       context.Context
	index     string
	schema    map[string]string
	algorithm string
	dim       int
}

type VectorSearchResult struct {
	Key   string                 `json:"key"`
	Score string                 `json:"score"`
	Props map[string]interface{} `json:"props"`
}

func NewVectorSearch(client rueidis.Client, ctx context.Context, index string, schema map[string]string, algoritm string, dim int) *VectorSearch {
	return &VectorSearch{client, ctx, index, schema, algoritm, dim}
}

func (vs *VectorSearch) Init() error {
	builder := vs.client.B().Arbitrary("FT.CREATE", vs.index, "ON", "HASH", "PREFIX", "1", "vector:", "SCHEMA")
	for k, v := range vs.schema {
		if v == "TAG" {
			builder.Args(k, v, "SEPARATOR", ";")
		} else {
			builder.Args(k, v)
		}
	}
	switch vs.algorithm {
	case "FLAT":
		builder.Args("v", "VECTOR", "FLAT", "6", "TYPE", "FLOAT32", "DIM", fmt.Sprint(vs.dim), "DISTANCE_METRIC", "L2")
	case "HNSW":
		builder.Args("v", "VECTOR", "HNSW", "16", "TYPE", "FLOAT32", "DIM", fmt.Sprint(vs.dim), "DISTANCE_METRIC", "L2", "INITIAL_CAP", "10000", "M", "40", "EF_CONSTRUCTION", "250", "EF_RUNTIME", "20", "EPSILON", "0.8")
	default:
		panic("unsupported algorithm")
	}

	return vs.client.Do(vs.ctx, builder.Build()).Error()
}

func (vs *VectorSearch) Add(key string, vector []float32, properties map[string]string) error {
	builder := vs.client.B().Hset().Key(fmt.Sprintf("vector:%s", key)).FieldValue().
		FieldValue("v", rueidis.VectorString32(vector))
	for k, v := range properties {
		builder.FieldValue(k, v)
	}
	return vs.client.Do(vs.ctx, builder.Build()).Error()
}

func (vs *VectorSearch) Search(n int, vector []float32, useFields []string, tags []string) ([]VectorSearchResult, error) {
	var query string
	if tags != nil {
		tagBuilder := "@tags:{"
		for _, tag := range tags {
			tagBuilder += tag + " | "
		}
		tagBuilder = strings.TrimSuffix(tagBuilder, " | ") + "}"
		query = fmt.Sprintf("%s=>[KNN %d @v $V]", tagBuilder, n)
	} else {
		query = fmt.Sprintf("[KNN %d @v $V]", n)
	}

	resp, err := vs.client.Do(vs.ctx, vs.client.B().FtSearch().Index(vs.index).
		Query(query).Params().Nargs(2).
		NameValue().NameValue("V", rueidis.VectorString32(vector)).Dialect(2).Build()).ToArray()
	if err != nil {
		return nil, err
	}

	var results []VectorSearchResult
	for i := 1; i < len(resp[1:]); i += 2 {
		key, _ := resp[i].ToString()
		props, _ := resp[i+1].AsStrMap()

		result := VectorSearchResult{
			Key:   key,
			Score: props["__v_score"],
			Props: make(map[string]interface{}),
		}

		for _, field := range useFields {
			if value, ok := props[field]; ok {
				result.Props[field] = value
			}
		}

		results = append(results, result)
	}
	return results, nil
}

func (vs *VectorSearch) Delete(key string) error {
	return vs.client.Do(vs.ctx, vs.client.B().Del().Key(key).Build()).Error()
}

func main() {
	client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{"127.0.0.1:6379"}})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	schema := map[string]string{
		"timestamp": "NUMERIC",
		"title":     "TEXT",
		"tags":      "TAG",
	}

	vs := NewVectorSearch(client, ctx, "idx", schema, "HNSW", DIM)
	if err := vs.Init(); err != nil {
		panic(err)
	}

	ts := strconv.Itoa(int(time.Now().Unix()))
	if err := vs.Add("a", generateRandomVector(DIM), map[string]string{"title": "Matrix", "timestamp": ts, "tags": "blue;green"}); err != nil {
		panic(err)
	}

	ts = strconv.Itoa(int(time.Now().Unix()))
	if err := vs.Add("b", generateRandomVector(DIM), map[string]string{"title": "Matrix 2", "timestamp": ts, "tags": "black;pink"}); err != nil {
		panic(err)
	}

	resp, err := vs.Search(5, generateRandomVector(DIM), []string{"title", "timestamp"}, []string{"pink"})
	if err != nil {
		panic(err)
	}

	// if err := vs.Delete("matrix"); err != nil {
	// 	panic(err)
	// }
	fmt.Println(resp)
}

func generateRandomVector(dim int) []float32 {
	vector := make([]float32, dim)
	for i := 0; i < len(vector); i++ {
		vector[i] = rand.Float32()
	}
	return vector
}
