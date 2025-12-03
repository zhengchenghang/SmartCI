package metrics

import (
	"fmt"
	"strings"
	"time"
)

// FormatDuration 格式化时长
func FormatDuration(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	
	if duration < time.Minute {
		return fmt.Sprintf("%.2f秒", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.1f分钟", duration.Minutes())
	} else {
		return fmt.Sprintf("%.1f小时", duration.Hours())
	}
}

// FormatTime 格式化时间
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// DisplayLatestExecution 显示最近一次执行信息
func DisplayLatestExecution(metadata *TaskMetadata) string {
	var sb strings.Builder
	
	sb.WriteString("╔════════════════════════════════════════════════════════════════\n")
	sb.WriteString("║ 最近一次执行记录\n")
	sb.WriteString("╠════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("║ 任务名称: %s\n", metadata.TaskName))
	sb.WriteString(fmt.Sprintf("║ 任务ID: %s\n", metadata.TaskID))
	sb.WriteString(fmt.Sprintf("║ 任务类型: %s\n", metadata.TaskType))
	sb.WriteString("╠────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("║ 开始时间: %s\n", FormatTime(metadata.StartTime)))
	sb.WriteString(fmt.Sprintf("║ 结束时间: %s\n", FormatTime(metadata.EndTime)))
	sb.WriteString(fmt.Sprintf("║ 执行时长: %s\n", FormatDuration(metadata.Duration)))
	sb.WriteString("╠────────────────────────────────────────────────────────────────\n")
	
	statusIcon := "✅"
	if metadata.Status != "success" {
		statusIcon = "❌"
	}
	sb.WriteString(fmt.Sprintf("║ 执行状态: %s %s\n", statusIcon, metadata.Status))
	
	if metadata.Error != "" {
		sb.WriteString(fmt.Sprintf("║ 错误信息: %s\n", metadata.Error))
	}
	
	sb.WriteString("╠────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("║ 任务目录: %s\n", metadata.TaskDir))
	sb.WriteString(fmt.Sprintf("║ 日志文件: %s\n", metadata.LogFile))
	sb.WriteString("╚════════════════════════════════════════════════════════════════\n")
	
	return sb.String()
}

// DisplayExecutionList 显示执行列表
func DisplayExecutionList(executions []*TaskMetadata, taskName string) string {
	var sb strings.Builder
	
	sb.WriteString("╔════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("║ 任务执行历史: %s (共 %d 条记录)\n", taskName, len(executions)))
	sb.WriteString("╠════════════════════════════════════════════════════════════════\n")
	sb.WriteString("║ 序号 │ 开始时间           │ 时长      │ 状态 │ 任务ID\n")
	sb.WriteString("╠════════════════════════════════════════════════════════════════\n")
	
	for i, exec := range executions {
		statusIcon := "✅"
		if exec.Status != "success" {
			statusIcon = "❌"
		}
		
		sb.WriteString(fmt.Sprintf("║ %-4d │ %s │ %-9s │ %s  │ %s\n",
			i+1,
			FormatTime(exec.StartTime),
			FormatDuration(exec.Duration),
			statusIcon,
			exec.TaskID,
		))
		
		if exec.Error != "" {
			// 截断错误信息以适应显示
			errorMsg := exec.Error
			if len(errorMsg) > 60 {
				errorMsg = errorMsg[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("║      │ 错误: %s\n", errorMsg))
		}
	}
	
	sb.WriteString("╚════════════════════════════════════════════════════════════════\n")
	
	return sb.String()
}

// DisplayStatistics 显示统计信息
func DisplayStatistics(stats *TaskStatistics, hours, days int) string {
	var sb strings.Builder
	
	timeRange := "全部时间"
	if hours > 0 || days > 0 {
		if days > 0 && hours > 0 {
			timeRange = fmt.Sprintf("最近 %d 天 %d 小时", days, hours)
		} else if days > 0 {
			timeRange = fmt.Sprintf("最近 %d 天", days)
		} else {
			timeRange = fmt.Sprintf("最近 %d 小时", hours)
		}
	}
	
	sb.WriteString("╔════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("║ 任务统计: %s (%s)\n", stats.TaskName, timeRange))
	sb.WriteString("╠════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("║ 总执行次数: %d 次\n", stats.TotalCount))
	sb.WriteString(fmt.Sprintf("║ 成功次数: ✅ %d 次\n", stats.SuccessCount))
	sb.WriteString(fmt.Sprintf("║ 失败次数: ❌ %d 次\n", stats.FailureCount))
	sb.WriteString(fmt.Sprintf("║ 成功率: %.2f%%\n", stats.SuccessRate))
	sb.WriteString("╠────────────────────────────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("║ 平均执行时长: %s\n", FormatDuration(stats.AvgDuration)))
	sb.WriteString(fmt.Sprintf("║ 最短执行时长: %s\n", FormatDuration(stats.MinDuration)))
	sb.WriteString(fmt.Sprintf("║ 最长执行时长: %s\n", FormatDuration(stats.MaxDuration)))
	sb.WriteString("╠────────────────────────────────────────────────────────────────\n")
	
	if stats.LastExecution != nil {
		lastStatusIcon := "✅"
		if stats.LastExecution.Status != "success" {
			lastStatusIcon = "❌"
		}
		sb.WriteString(fmt.Sprintf("║ 最近一次执行: %s %s\n", 
			FormatTime(stats.LastExecution.StartTime), lastStatusIcon))
	}
	
	if stats.FirstExecution != nil {
		sb.WriteString(fmt.Sprintf("║ 最早一次执行: %s\n", 
			FormatTime(stats.FirstExecution.StartTime)))
	}
	
	sb.WriteString("╚════════════════════════════════════════════════════════════════\n")
	
	return sb.String()
}

// DisplayAllTasksSummary 显示所有任务的简要统计
func DisplayAllTasksSummary(allMetadata []*TaskMetadata) string {
	var sb strings.Builder
	
	// 统计每个任务的执行次数
	taskStats := make(map[string]*TaskStatistics)
	
	for _, metadata := range allMetadata {
		taskName := metadata.TaskName
		if _, exists := taskStats[taskName]; !exists {
			taskStats[taskName] = &TaskStatistics{
				TaskName:       taskName,
				LastExecution:  metadata,
				FirstExecution: metadata,
			}
		}
		
		stats := taskStats[taskName]
		stats.TotalCount++
		
		if metadata.Status == "success" {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
		
		// 更新最早执行时间
		if metadata.StartTime.Before(stats.FirstExecution.StartTime) {
			stats.FirstExecution = metadata
		}
	}
	
	// 计算成功率
	for _, stats := range taskStats {
		if stats.TotalCount > 0 {
			stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
		}
	}
	
	sb.WriteString("╔════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("║ 所有任务统计 (共 %d 个任务)\n", len(taskStats)))
	sb.WriteString("╠════════════════════════════════════════════════════════════════\n")
	sb.WriteString("║ 任务名称               │ 执行次数 │ 成功率   │ 最近执行\n")
	sb.WriteString("╠════════════════════════════════════════════════════════════════\n")
	
	// 按任务名称排序
	var taskNames []string
	for taskName := range taskStats {
		taskNames = append(taskNames, taskName)
	}
	
	for _, taskName := range taskNames {
		stats := taskStats[taskName]
		lastExecTime := ""
		if stats.LastExecution != nil {
			lastExecTime = stats.LastExecution.StartTime.Format("01-02 15:04")
		}
		
		sb.WriteString(fmt.Sprintf("║ %-22s │ %-8d │ %6.2f%% │ %s\n",
			truncateString(taskName, 22),
			stats.TotalCount,
			stats.SuccessRate,
			lastExecTime,
		))
	}
	
	sb.WriteString("╚════════════════════════════════════════════════════════════════\n")
	
	return sb.String()
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
