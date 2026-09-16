// 本文件定义跨协议生成参数，避免请求解码后丢失调用方设置。
package llm

import (
	"errors"
	"math"
)

// GenerationOptions 保存显式生成设置；指针用于区分未设置和合法的零值。
type GenerationOptions struct {
	// ReasoningEffort 是调用方选择的离散思考档位；空值保留模型默认档位。
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// MaxOutputTokens 是输出 token 预算；nil 保留适配器默认值。
	MaxOutputTokens *int `json:"max_output_tokens,omitempty"`
	// Temperature 是采样温度；零值必须传给上游。
	Temperature *float64 `json:"temperature,omitempty"`
	// TopP 是概率质量采样阈值；nil 保留适配器默认值。
	TopP *float64 `json:"top_p,omitempty"`
	// TopK 是候选 token 数；nil 保留适配器默认值。
	TopK *int `json:"top_k,omitempty"`
}

// Validate 拒绝非法数值，防止负整数转换为上游无符号数后溢出。
func (options GenerationOptions) Validate() error {
	if options.MaxOutputTokens != nil && *options.MaxOutputTokens <= 0 {
		return errors.New("max output tokens must be positive")
	}
	if options.Temperature != nil && (!finite(*options.Temperature) || *options.Temperature < 0 || *options.Temperature > 2) {
		return errors.New("temperature must be between 0 and 2")
	}
	if options.TopP != nil && (!finite(*options.TopP) || *options.TopP < 0 || *options.TopP > 1) {
		return errors.New("top_p must be between 0 and 1")
	}
	if options.TopK != nil && *options.TopK < 0 {
		return errors.New("top_k cannot be negative")
	}
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
