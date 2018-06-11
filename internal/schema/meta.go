//go:generate go-bindata -ignore=\.go -pkg=schema -o=bindata.go ./...
package schema

var Meta *Schema

func init() {
	Meta = &Schema{} //bootstrap
	Meta = New()
	schemaBytes, _ := metaGraphqlBytes()
	if err := Meta.Parse(string(schemaBytes)); err != nil {
		panic(err)
	}
}
