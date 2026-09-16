// 本文件验证真实协议解码后的并行工具历史和参数在上游请求中保持正确语义。
package devin

import (
	"strings"
	"testing"

	"github.com/leookun/devin-2api/internal/api/openai/chat"
	"github.com/leookun/devin-2api/internal/api/openai/responses"
	"github.com/leookun/devin-2api/internal/llm"
)

func TestResponsesParallelToolsStayInOneAssistantTurn(t *testing.T) {
	decoded, err := responses.DecodeRequest([]byte(`{
		"model":"test", "input":[
			{"role":"user","content":"read both"},
			{"role":"assistant","content":"Reading both files."},
			{"type":"function_call","call_id":"one","name":"read","arguments":"{\"path\":\"a\"}"},
			{"type":"function_call","call_id":"two","name":"read","arguments":"{\"path\":\"b\"}"},
			{"type":"function_call_output","call_id":"two","output":"B"},
			{"type":"function_call_output","call_id":"one","output":"A"},
			{"type":"function_call","call_id":"three","name":"read","arguments":"{}"},
			{"type":"function_call_output","call_id":"three","output":"C"},
			{"role":"user","content":"next round"},
			{"role":"assistant","content":"Done."}
		]}`))
	if err != nil {
		t.Fatal(err)
	}
	converted, err := buildRequest(decoded.Context, Config{Token: "test", Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	prompts := converted.GetChatMessagePrompts()
	if len(prompts) != 8 {
		t.Fatalf("got %d prompts, want 8", len(prompts))
	}
	assistant := prompts[1]
	if assistant.GetPrompt() != "Reading both files." || len(assistant.GetToolCalls()) != 2 {
		t.Fatalf("assistant turn was split: %v", assistant)
	}
	if assistant.GetToolCalls()[0].GetId() != "one" || assistant.GetToolCalls()[1].GetId() != "two" {
		t.Fatal("tool order changed")
	}
	if prompts[2].GetToolCallId() != "two" || prompts[3].GetToolCallId() != "one" {
		t.Fatal("tool result associations changed")
	}
	if len(prompts[4].GetToolCalls()) != 1 || prompts[4].GetToolCalls()[0].GetId() != "three" {
		t.Fatal("merged across a tool-result boundary")
	}
	if prompts[7].GetPrompt() != "Done." {
		t.Fatal("merged across a user boundary")
	}
}

func TestChatGenerationAndReasoningReachDevin(t *testing.T) {
	decoded, err := chat.DecodeRequest([]byte(`{
		"model":"test","max_tokens":777,"max_completion_tokens":321,"temperature":0,"top_p":0.7,
		"messages":[
			{"role":"user","content":"read"},
			{"role":"assistant","content":null,"reasoning_content":"Need both files.","tool_calls":[
				{"id":"one","type":"function","function":{"name":"read","arguments":"{}"}},
				{"id":"two","type":"function","function":{"name":"read","arguments":"{}"}}
			]},
			{"role":"tool","tool_call_id":"one","content":"A"},
			{"role":"tool","tool_call_id":"two","content":"B"}
		]}`))
	if err != nil {
		t.Fatal(err)
	}
	topK := 12
	decoded.Context.Generation.TopK = &topK
	converted, err := buildRequest(decoded.Context, Config{Token: "test", Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	settings := converted.GetConfiguration()
	if settings.GetMaxTokens() != 321 || settings.Temperature == nil || settings.GetTemperature() != 0 || settings.GetTopP() != 0.7 || settings.GetTopK() != 12 {
		t.Fatalf("incorrect generation settings: %v", settings)
	}
	assistant := converted.GetChatMessagePrompts()[1]
	if assistant.GetThinking() != "Need both files." || len(assistant.GetToolCalls()) != 2 {
		t.Fatal("reasoning or parallel calls lost")
	}
	defaults, err := buildRequest(llm.RequestMessages{}, Config{Token: "test", Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if defaults.GetConfiguration().GetMaxTokens() != 128000 || defaults.GetConfiguration().GetTemperature() != 1 {
		t.Fatal("omitted options changed defaults")
	}
}

func TestUnsupportedStateAndInvalidToolArgumentsAreRejected(t *testing.T) {
	_, err := responses.DecodeRequest([]byte(`{"model":"test","input":"continue","previous_response_id":"previous"}`))
	if err == nil || !strings.Contains(err.Error(), "full conversation") {
		t.Fatalf("unexpected state error: %v", err)
	}
	_, err = chat.DecodeRequest([]byte(`{"model":"test","messages":[{"role":"assistant","tool_calls":[{"id":"one","function":{"name":"read","arguments":"[1]"}}]}]}`))
	if err == nil {
		t.Fatal("invalid tool arguments silently replaced with an empty object")
	}
}
