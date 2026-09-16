// 本文件防止三种 HTTP 协议的生成参数在进入适配器前丢失。
package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leookun/devin-2api/internal/config"
	"github.com/leookun/devin-2api/internal/llm"
)

func TestGenerationOptionsReachAdapter(t *testing.T) {
	for endpoint, body := range map[string]string{
		"/v1/responses":        `{"model":"test","input":"hi","max_output_tokens":321,"temperature":0,"top_p":0.7}`,
		"/v1/chat/completions": `{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":999,"max_completion_tokens":321,"temperature":0,"top_p":0.7}`,
		"/v1/messages":         `{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":321,"temperature":0,"top_p":0.7,"top_k":12}`,
	} {
		t.Run(endpoint, func(t *testing.T) {
			fake := &fakeAdapter{events: []llm.ResponseEvent{{Type: llm.ResponseEventDone, Reason: llm.StopReasonStop, Message: &llm.AssistantMessage{StopReason: llm.StopReasonStop}}}}
			application := New(fake, config.ServerConfig{Listen: ":0"}, nil)
			response := httptest.NewRecorder()
			application.Router().ServeHTTP(response, httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(body)))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", response.Code, response.Body.String())
			}
			options := fake.lastRequest.Generation
			if options.MaxOutputTokens == nil || *options.MaxOutputTokens != 321 || options.Temperature == nil || *options.Temperature != 0 || options.TopP == nil || *options.TopP != 0.7 {
				t.Fatalf("generation settings lost: %+v", options)
			}
			if endpoint == "/v1/messages" && (options.TopK == nil || *options.TopK != 12) {
				t.Fatal("top_k lost")
			}
		})
	}
}

func TestInvalidGenerationOptionsReturnBadRequest(t *testing.T) {
	for _, field := range []string{`"max_tokens":-1`, `"max_tokens":0`, `"temperature":-0.1`, `"temperature":3`, `"top_p":1.1`} {
		fake := &fakeAdapter{}
		application := New(fake, config.ServerConfig{Listen: ":0"}, nil)
		body := `{"model":"test","messages":[{"role":"user","content":"hi"}],` + field + `}`
		response := httptest.NewRecorder()
		application.Router().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest || fake.lastRequest.Model != "" {
			t.Fatalf("%s: status=%d, adapter model=%q", field, response.Code, fake.lastRequest.Model)
		}
	}
}
