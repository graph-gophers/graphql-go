package autogen

import (
	"fmt"
	"testing"
	"time"
)

func TestGenFile(t *testing.T) {
	err := GenFile("/Users/luoxiaomin/go/src/github.com/ricklxm/graphql-go/autogen/all.graphql",
		"/Users/luoxiaomin/go/src/github.com/ricklxm/graphql-go/autogen/generated/", "generated")
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(1000)
}
