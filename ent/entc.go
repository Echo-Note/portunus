//go:build ignore
// +build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

func main() {
	err := entc.Generate("./ent/schema",
		&gen.Config{
			Target:  "./ent/generated",
			Package: "github.com/Echo-Note/portunus/ent/generated",
			Features: []gen.Feature{
				gen.FeatureExecQuery,  // 支持原生 SQL
				gen.FeatureUpsert,    // 支持 UPSERT
				gen.FeatureModifier,  // 支持 UpdateOne().SetXxx() 修改器
				gen.FeatureLock,      // 支持 SELECT ... FOR UPDATE
				gen.FeatureIntercept, // 支持拦截器（中间件）
			},
		},
	)
	if err != nil {
		log.Fatalf("running ent codegen: %v", err)
	}
}
