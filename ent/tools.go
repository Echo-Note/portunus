//go:build tools
// +build tools

// Package tools 管理开发工具依赖，确保 go mod tidy 不会移除 entc 和 atlas 的间接依赖。
package tools

import (
	_ "ariga.io/atlas/sql/migrate"
	_ "ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/postgres"
	_ "ariga.io/atlas/sql/schema"
	_ "ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"
	_ "ariga.io/atlas/sql/sqltool"
	_ "entgo.io/ent/entc"
	_ "entgo.io/ent/entc/gen"
	_ "github.com/go-openapi/inflect"
	_ "golang.org/x/tools/go/ast/astutil"
	_ "golang.org/x/tools/go/packages"
	_ "golang.org/x/tools/imports"
)