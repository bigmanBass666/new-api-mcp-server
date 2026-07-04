package hightools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/QuantumNous/new-api-mcp-server/internal/client"
	"github.com/QuantumNous/new-api-mcp-server/internal/observability"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracerTestAndReport = otel.Tracer("hightools.test_and_report")

// testAndReportConfig allows tests to override poll timing.
type testAndReportConfig struct {
	pollInterval time.Duration
	pollTimeout  time.Duration
}

var defaultTestAndReportConfig = testAndReportConfig{
	pollInterval: 2 * time.Second,
	pollTimeout:  120 * time.Second,
}

// channelTestResponse maps the upstream POST/GET /api/channel/test response.
type channelTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    *struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	} `json:"data"`
}

// systemTaskResponse maps the upstream GET /api/system-task/{task_id} response.
type systemTaskResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    *struct {
		TaskID string      `json:"task_id"`
		Type   string      `json:"type"`
		Status string      `json:"status"`
		Result interface{} `json:"result"`
		Error  string      `json:"error"`
	} `json:"data"`
}

// channelTestSummary mirrors the upstream channelTestSummary struct.
type channelTestSummary struct {
	Tested    int `json:"tested"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Disabled  int `json:"disabled"`
	Enabled   int `json:"enabled"`
}

// NewTestAndReportTool returns a ToolDef for triggering a full channel test
// and returning a formatted health summary.
func NewTestAndReportTool(c *client.Client, metrics *observability.Metrics) ToolDef {
	return ToolDef{
		Name:        "test_and_report",
		Description: "Test all channels and return a health summary. Triggers an async full-channel test, waits for completion, and returns a formatted report.",
		InputSchema: inputSchemaTestAndReport(),
		Handler:    handleTestAndReport(c, metrics),
	}
}

func inputSchemaTestAndReport() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func handleTestAndReport(c *client.Client, metrics *observability.Metrics) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, span := tracerTestAndReport.Start(ctx, "test_and_report",
			trace.WithAttributes(
				attribute.String("tool", "test_and_report"),
			),
		)
		defer span.End()

		start := time.Now()
		status := "success"
		defer func() {
			if metrics != nil {
				metrics.ToolRequestsTotal.WithLabelValues("test_and_report", status).Inc()
				metrics.ToolRequestDuration.WithLabelValues("test_and_report").Observe(time.Since(start).Seconds())
			}
		}()

		// Step 1: Trigger the channel test
		result, err := triggerChannelTest(ctx, c, metrics)
		if err != nil {
			status = "error"
			return errorResultTestAndReport(fmt.Sprintf("failed to trigger channel test: %v", err)), nil
		}
		if !result.Success {
			status = "error"
			// If another test is already running, return a friendly message
			if result.Message != "" {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Channel test already in progress: %s", result.Message)},
					},
				}, nil
			}
			return errorResultTestAndReport(fmt.Sprintf("channel test trigger failed: %s", result.Message)), nil
		}

		taskID := ""
		if result.Data != nil {
			taskID = result.Data.TaskID
		}
		if taskID == "" {
			status = "error"
			return errorResultTestAndReport("channel test did not return a task_id"), nil
		}

		slog.InfoContext(ctx, "channel test triggered", "task_id", taskID)

		// Step 2: Poll for completion
		summary, pollErr := pollTaskCompletion(ctx, c, taskID, defaultTestAndReportConfig)

		elapsed := time.Since(start).Seconds()

		// Step 3: Format output
		output := formatTestReport(summary, elapsed, pollErr)

		slog.InfoContext(ctx, "test_and_report completed",
			"task_id", taskID,
			"elapsed_s", elapsed,
			"tested", summary.Tested,
			"succeeded", summary.Succeeded,
			"failed", summary.Failed,
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output},
			},
		}, nil
	}
}

// triggerChannelTest calls GET /api/channel/test to enqueue a channel test task.
func triggerChannelTest(ctx context.Context, c *client.Client, metrics *observability.Metrics) (*channelTestResponse, error) {
	resp, err := c.Do(ctx, client.SourceAPI, "GET", "/api/channel/test", nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("upstream request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if metrics != nil {
		metrics.UpstreamRequestsTotal.WithLabelValues("GET", "/api/channel/test", fmt.Sprintf("%d", resp.StatusCode)).Inc()
	}

	var result channelTestResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}

// pollTaskCompletion polls GET /api/system-task/{task_id} until the task
// reaches a terminal state or the timeout expires.
func pollTaskCompletion(ctx context.Context, c *client.Client, taskID string, cfg testAndReportConfig) (channelTestSummary, error) {
	deadline := time.Now().Add(cfg.pollTimeout)

	for time.Now().Before(deadline) {
		// Check context cancellation
		if ctx.Err() != nil {
			return channelTestSummary{}, fmt.Errorf("context cancelled: %w", ctx.Err())
		}

		resp, err := c.Do(ctx, client.SourceAPI, "GET", "/api/system-task/"+taskID, nil, nil, nil)
		if err != nil {
			return channelTestSummary{}, fmt.Errorf("poll request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return channelTestSummary{}, fmt.Errorf("poll read: %w", err)
		}

		var taskResp systemTaskResponse
		if err := json.Unmarshal(body, &taskResp); err != nil {
			return channelTestSummary{}, fmt.Errorf("poll parse: %w", err)
		}

		if !taskResp.Success || taskResp.Data == nil {
			slog.Warn("unexpected poll response", "body", string(body))
			time.Sleep(cfg.pollInterval)
			continue
		}

		switch taskResp.Data.Status {
		case "succeeded":
			if taskResp.Data.Result == nil {
				return channelTestSummary{}, nil
			}
			// result can be a map[string]interface{} from JSON decoding
			summary, err := decodeSummary(taskResp.Data.Result)
			if err != nil {
				return channelTestSummary{}, fmt.Errorf("decode result: %w", err)
			}
			return summary, nil

		case "failed":
			errMsg := taskResp.Data.Error
			if errMsg == "" {
				errMsg = "channel test task failed"
			}
			return channelTestSummary{}, fmt.Errorf("%s", errMsg)

		case "pending", "running":
			// Continue polling
			time.Sleep(cfg.pollInterval)

		default:
			slog.Warn("unexpected task status", "status", taskResp.Data.Status)
			time.Sleep(cfg.pollInterval)
		}
	}

	return channelTestSummary{}, fmt.Errorf("timeout after waiting %v", cfg.pollTimeout)
}

// decodeSummary converts the raw result (from JSON decoding) to channelTestSummary.
func decodeSummary(raw interface{}) (channelTestSummary, error) {
	// If it's already a map, marshal and unmarshal to the struct
	data, err := json.Marshal(raw)
	if err != nil {
		return channelTestSummary{}, fmt.Errorf("marshal result: %w", err)
	}
	var summary channelTestSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return channelTestSummary{}, fmt.Errorf("unmarshal summary: %w", err)
	}
	return summary, nil
}

// formatTestReport produces a formatted text summary.
func formatTestReport(summary channelTestSummary, elapsed float64, pollErr error) string {
	if pollErr != nil {
		// Partial summary on timeout/error
		return fmt.Sprintf(`# 渠道测试报告

## 概览
- 测试数: %d
- 通过: %d
- 失败: %d
- 已禁用: %d
- 已启用: %d
- 耗时: %.1fs

## 测试状态: ⏳ 未完成（%s）
`, summary.Tested, summary.Succeeded, summary.Failed, summary.Disabled, summary.Enabled, elapsed, pollErr.Error())
	}

	status := "✅ 已完成"
	if summary.Failed > 0 {
		status = "⚠️ 已完成（有失败）"
	}

	report := fmt.Sprintf(`# 渠道测试报告

## 概览
- 测试数: %d
- 通过: %d
- 失败: %d
- 已禁用: %d
- 已启用: %d
- 耗时: %.1fs

## 测试状态: %s
`, summary.Tested, summary.Succeeded, summary.Failed, summary.Disabled, summary.Enabled, elapsed, status)

	if summary.Failed > 0 {
		report += "\n## 失败渠道详情\n"
		report += fmt.Sprintf("- %d 个渠道测试未通过。请调用 `api_search_channels` 或 `api_get_all_channels` 查看具体渠道状态。\n", summary.Failed)
	}

	return report
}

func errorResultTestAndReport(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}