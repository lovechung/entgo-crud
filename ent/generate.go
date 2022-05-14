package ent

// 1.添加特性开关
// 2.开启模板生成（建议将模板文件放在 ent/template 中）
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature sql/execquery,sql/modifier --template "./template/template.go.tmpl" ./schema
