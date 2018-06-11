//go:generate go-bindata -ignore=\.go -pkg=schema -o=bindata.go ./...
package schema

import "bytes"

var Meta *Schema

func init() {
	Meta = &Schema{}
	Meta = New()
	if err := Meta.Parse(metaSchemaString()); err != nil {
		panic(err)
	}
}

func metaSchemaString() string {
	buf := bytes.Buffer{}
	for _, name := range AssetNames() {
		b := MustAsset(name)
		buf.Write(b)

		if len(b) > 0 && b[len(b)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}
