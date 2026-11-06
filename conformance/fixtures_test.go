package conformance_test

import (
	"os"
	"testing"

	hubv1 "github.com/reinsman/proto/gen/go/rein/hub/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// fixtures 必须能由生成类型解析，未知字段不能悄悄通过。
func TestFixtures(t *testing.T) {
	fixtures := map[string]proto.Message{
		"execution-event.json": &hubv1.ExecutionEvent{},
		"cancel-command.json":  &hubv1.ControlCommand{},
		"hello.json":           &hubv1.ConnectRequest{},
	}
	for name, message := range fixtures {
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile("../fixtures/" + name)
			if err != nil {
				t.Fatal(err)
			}
			if err := protojson.Unmarshal(body, message); err != nil {
				t.Fatal(err)
			}
			encoded, err := proto.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			decoded := message.ProtoReflect().New().Interface()
			if err := proto.Unmarshal(encoded, decoded); err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(message, decoded) {
				t.Fatal("binary roundtrip changed fixture")
			}
			if name == "execution-event.json" {
				golden, err := os.ReadFile("../fixtures/execution-event.bin")
				if err != nil {
					t.Fatal(err)
				}
				if err := proto.Unmarshal(golden, decoded); err != nil {
					t.Fatal(err)
				}
				if !proto.Equal(message, decoded) {
					t.Fatal("golden binary differs from JSON fixture")
				}
			}
		})
	}
}
