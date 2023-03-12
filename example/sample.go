//go:generate go run ../cmd --source=$GOFILE --name="claim"
package example

import "github.com/guregu/dynamo"

type XClaim struct {
	aa  string
	bbb string
	sss string
}
type XClaimTable struct {
	dynamo.Table
}

func NewXClaimTable(d dynamo.DB) XClaimTable {
	return XClaimTable{d.Table("claim")}
}
