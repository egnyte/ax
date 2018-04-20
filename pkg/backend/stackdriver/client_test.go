package stackdriver

import (
	"fmt"
	"testing"
)

func TestAttributeDecoding(t *testing.T) {
	exampleJson := []byte(`{"fields":{"age":{"Kind":{"NumberValue":34}},"bool":{"Kind":{"BoolValue":true}},"list":{"Kind":{"ListValue":{"values":[{"Kind":{"NumberValue":1}},{"Kind":{"NumberValue":2}},{"Kind":{"NumberValue":3}},{"Kind":{"NumberValue":4}}]}}},"name":{"Kind":{"StringValue":"test"}},"obj":{"Kind":{"StructValue":{"fields":{"name":{"Kind":{"StringValue":"test"}}}}}},"slist":{"Kind":{"ListValue":{"values":[{"Kind":{"StringValue":"aap"}},{"Kind":{"StringValue":"noot"}},{"Kind":{"StringValue":"mies"}}]}}}}}`)
	m := payloadToAttributes(exampleJson)
	fmt.Printf("%+v\n", m)
	if m["age"] != int64(34) {
		t.Error("age")
	}
	if m["bool"] != true {
		t.Error("bool")
	}
	if len(m["list"].([]interface{})) != 4 {
		t.Error("list")
	}
	if len(m["slist"].([]interface{})) != 3 {
		t.Error("slist")
	}
	obj, ok := m["obj"].(map[string]interface{})
	if !ok {
		t.Error("obj")
	}

	if obj["name"] != "test" {
		t.Error("obj.name")
	}
}
