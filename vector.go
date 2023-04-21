package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rueian/rueidis"
)

type VectorSearch struct {
	driver     string
	host       []string
	ctx        context.Context
	collection string
	schema     map[string]string
	algorithm  string
	dim        int
	client     interface{}
}

type VectorSearchResult struct {
	Key   string                 `json:"key"`
	Score string                 `json:"score"`
	Props map[string]interface{} `json:"props"`
}

func NewSchema(schema map[string]string) map[string]string {
	if _, ok := schema["tags"]; !ok {
		schema["tags"] = "TAG"
	}
	if _, ok := schema["timestamp"]; !ok {
		schema["timestamp"] = "NUMERIC"
	}
	return schema
}

func NewVectorSearch(driver string, host []string, ctx context.Context, index string, schema map[string]string, algorithm string, dim int) *VectorSearch {
	if driver == "redis" {
		client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: host})
		if err != nil {
			panic(err)
		}
		return &VectorSearch{
			driver:     driver,
			host:       host,
			ctx:        ctx,
			collection: index,
			schema:     NewSchema(schema),
			algorithm:  algorithm,
			dim:        dim,
			client:     client,
		}
	}

	return &VectorSearch{}
}

func (vs *VectorSearch) CreateCollection() error {

	if vs.driver == "redis" {
		client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: vs.host})
		if err != nil {
			return err
		}
		builder := client.B().Arbitrary("FT.CREATE", vs.collection, "ON", "HASH", "PREFIX", "1", vs.collection+":", "SCHEMA")
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
		err = client.Do(vs.ctx, builder.Build()).Error()
		if err != nil {
			return err
		}
		vs.client = client
	}

	return nil
}

func (vs *VectorSearch) Add(key string, vector []float32, properties map[string]string) error {
	if c, ok := vs.client.(rueidis.Client); ok {
		builder := c.B().Hset().Key(fmt.Sprintf("%s:%s", vs.collection, key)).FieldValue().
			FieldValue("v", rueidis.VectorString32(vector))
		for k, v := range properties {
			builder.FieldValue(k, v)
		}
		return c.Do(vs.ctx, builder.Build()).Error()
	}
	return nil
}

func (vs *VectorSearch) Search(k int, vector []float32, returnFields []string, tags []string) ([]VectorSearchResult, error) {
	if c, ok := vs.client.(rueidis.Client); ok {
		var query string
		if tags != nil {
			tagBuilder := "@tags:{"
			for _, tag := range tags {
				tagBuilder += tag + " | "
			}
			tagBuilder = strings.TrimSuffix(tagBuilder, " | ") + "}"
			query = fmt.Sprintf("%s=>[KNN %d @v $V]", tagBuilder, k)
		} else {
			query = fmt.Sprintf("[KNN %d @v $V]", k)
		}

		resp, err := c.Do(vs.ctx, c.B().FtSearch().Index(vs.collection).
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

			for _, field := range returnFields {
				if value, ok := props[field]; ok {
					result.Props[field] = value
				}
			}

			results = append(results, result)
		}
	}
	return []VectorSearchResult{}, nil
}

func (vs *VectorSearch) Delete(key string) error {
	if c, ok := vs.client.(rueidis.Client); ok {
		return c.Do(vs.ctx, c.B().Del().Key(key).Build()).Error()
	}
	return nil
}
