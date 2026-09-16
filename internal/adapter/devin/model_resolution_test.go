// 本文件验证独立思考参数控制 SWE-2 路由，并保持客户端可见模型名稳定。
package devin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/leookun/devin-2api/internal/api/openai/chat"
	"github.com/leookun/devin-2api/internal/api/openai/responses"
	"github.com/leookun/devin-2api/internal/llm"
	"google.golang.org/protobuf/proto"
	devinproto "local/devinproto"
)

func TestSWE2ReasoningParameterSelectsUpstreamModel(t *testing.T) {
	for effort, want := range map[string]string{"": "swe-2-high", "medium": "swe-2-medium", "high": "swe-2-high", "max": "swe-2-max"} {
		t.Run(effort, func(t *testing.T) {
			chatBody, _ := json.Marshal(map[string]any{"model": "swe-2", "messages": []any{map[string]any{"role": "user", "content": "hi"}}, "reasoning_effort": effort})
			chatRequest, err := chat.DecodeRequest(chatBody)
			if err != nil {
				t.Fatal(err)
			}
			responsesBody, _ := json.Marshal(map[string]any{"model": "swe-2", "input": "hi", "reasoning": map[string]any{"effort": effort}})
			responsesRequest, err := responses.DecodeRequest(responsesBody)
			if err != nil {
				t.Fatal(err)
			}
			for _, request := range []llm.RequestMessages{chatRequest.Context, responsesRequest.Context} {
				converted, err := buildRequest(request, Config{Token: "test", Model: "swe-2"})
				if err != nil {
					t.Fatal(err)
				}
				if converted.GetChatModelUid() != want || request.Model != "swe-2" {
					t.Fatalf("effort %q: upstream=%q, public=%q", effort, converted.GetChatModelUid(), request.Model)
				}
			}
		})
	}
}

func TestSWE2InvalidEffortFailsBeforeUpstreamCall(t *testing.T) {
	// nil client 使意外发起 RPC 立即失败，证明无效档位在网络调用前被拒绝。
	a := &Adapter{config: Config{Model: "swe-2"}}
	for _, effort := range []string{"none", "low", "minimal", "xhigh", "auto", "unknown"} {
		_, err := a.Stream(context.Background(), llm.RequestMessages{Model: "swe-2", Generation: llm.GenerationOptions{ReasoningEffort: effort}})
		if err == nil || !strings.HasPrefix(err.Error(), "invalid_argument:") {
			t.Fatalf("effort %q: %v", effort, err)
		}
	}
	got, err := resolveModel("glm-5-3-high", "")
	if err != nil || got != "glm-5-3-high" {
		t.Fatal("unrelated model changed")
	}
	if _, err := resolveModel("glm-5-3-high", "max"); err == nil {
		t.Fatal("unimplemented model effort silently ignored")
	}
}

func TestSWE2ResponseKeepsUnifiedModelName(t *testing.T) {
	decoder := newResponseDecoder("swe-2")
	decoder.start()
	decoder.decode(&devinproto.GetChatMessageResponse{ActualModelUid: proto.String("swe-2-max"), DeltaText: proto.String("hello")})
	body, err := chat.EncodeResponse(&decoder.partial)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response["model"] != "swe-2" {
		t.Fatalf("model = %v", response["model"])
	}
}
