// 本文件把统一模型名与思考档位转换为已确认的 Devin 上游模型标识。
package devin

import "fmt"

// resolveModel 只映射已确认的模型族，避免用任意字符串拼接猜测上游能力。
func resolveModel(model, effort string) (string, error) {
	if model != "swe-2" {
		if effort != "" {
			return "", fmt.Errorf("invalid_argument: reasoning_effort is not supported for model %q", model)
		}
		return model, nil
	}
	switch effort {
	case "", "high":
		return "swe-2-high", nil
	case "medium":
		return "swe-2-medium", nil
	case "max":
		return "swe-2-max", nil
	default:
		return "", fmt.Errorf("invalid_argument: swe-2 reasoning_effort must be medium, high or max; got %q", effort)
	}
}
