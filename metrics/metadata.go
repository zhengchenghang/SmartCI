package metrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// TaskMetadata 任务执行元数据
type TaskMetadata struct {
	TaskID     string                 `json:"task_id"`      // 任务ID
	TaskName   string                 `json:"task_name"`    // 任务名称
	TaskType   string                 `json:"task_type"`    // 任务类型: bash/docker/repo
	StartTime  time.Time              `json:"start_time"`   // 开始时间
	EndTime    time.Time              `json:"end_time"`     // 结束时间
	Duration   float64                `json:"duration"`     // 执行时长（秒）
	Status     string                 `json:"status"`       // 执行状态: success/failure
	Error      string                 `json:"error"`        // 错误信息
	LogFile    string                 `json:"log_file"`     // 日志文件路径
	TaskDir    string                 `json:"task_dir"`     // 任务目录路径
	Config     map[string]interface{} `json:"config"`       // 任务配置（可选）
}

// SaveMetadata 保存任务元数据到任务目录
func SaveMetadata(metadata *TaskMetadata) error {
	if metadata.TaskDir == "" {
		return fmt.Errorf("任务目录不能为空")
	}

	metadataFile := filepath.Join(metadata.TaskDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %v", err)
	}

	err = ioutil.WriteFile(metadataFile, data, 0644)
	if err != nil {
		return fmt.Errorf("写入元数据文件失败: %v", err)
	}

	return nil
}

// LoadMetadata 从任务目录加载元数据
func LoadMetadata(taskDir string) (*TaskMetadata, error) {
	metadataFile := filepath.Join(taskDir, "metadata.json")
	
	data, err := ioutil.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("读取元数据文件失败: %v", err)
	}

	var metadata TaskMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, fmt.Errorf("解析元数据失败: %v", err)
	}

	return &metadata, nil
}

// ListAllMetadata 列出指定目录下所有任务的元数据
func ListAllMetadata(logDir string) ([]*TaskMetadata, error) {
	var metadataList []*TaskMetadata

	entries, err := ioutil.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("读取日志目录失败: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskDir := filepath.Join(logDir, entry.Name())
		metadataFile := filepath.Join(taskDir, "metadata.json")

		// 检查是否存在元数据文件
		if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
			continue
		}

		metadata, err := LoadMetadata(taskDir)
		if err != nil {
			// 忽略无法加载的元数据
			continue
		}

		metadataList = append(metadataList, metadata)
	}

	// 按开始时间倒序排序
	sort.Slice(metadataList, func(i, j int) bool {
		return metadataList[i].StartTime.After(metadataList[j].StartTime)
	})

	return metadataList, nil
}

// FilterMetadataByTaskName 按任务名称过滤元数据
func FilterMetadataByTaskName(metadataList []*TaskMetadata, taskName string) []*TaskMetadata {
	var filtered []*TaskMetadata
	for _, metadata := range metadataList {
		if metadata.TaskName == taskName {
			filtered = append(filtered, metadata)
		}
	}
	return filtered
}

// FilterMetadataByTimeRange 按时间范围过滤元数据
func FilterMetadataByTimeRange(metadataList []*TaskMetadata, start, end time.Time) []*TaskMetadata {
	var filtered []*TaskMetadata
	for _, metadata := range metadataList {
		if (metadata.StartTime.Equal(start) || metadata.StartTime.After(start)) &&
			(metadata.StartTime.Before(end) || metadata.StartTime.Equal(end)) {
			filtered = append(filtered, metadata)
		}
	}
	return filtered
}

// GetLatestExecution 获取指定任务的最近一次执行记录
func GetLatestExecution(logDir, taskName string) (*TaskMetadata, error) {
	allMetadata, err := ListAllMetadata(logDir)
	if err != nil {
		return nil, err
	}

	filtered := FilterMetadataByTaskName(allMetadata, taskName)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("未找到任务 '%s' 的执行记录", taskName)
	}

	return filtered[0], nil
}

// ListExecutions 列出指定任务在时间范围内的执行记录
func ListExecutions(logDir, taskName string, hours, days int) ([]*TaskMetadata, error) {
	allMetadata, err := ListAllMetadata(logDir)
	if err != nil {
		return nil, err
	}

	// 过滤任务名称
	filtered := FilterMetadataByTaskName(allMetadata, taskName)

	// 如果指定了时间范围，则进一步过滤
	if hours > 0 || days > 0 {
		end := time.Now()
		duration := time.Duration(days*24+hours) * time.Hour
		start := end.Add(-duration)
		filtered = FilterMetadataByTimeRange(filtered, start, end)
	}

	return filtered, nil
}

// TaskStatistics 任务统计信息
type TaskStatistics struct {
	TaskName       string        `json:"task_name"`       // 任务名称
	TotalCount     int           `json:"total_count"`     // 总执行次数
	SuccessCount   int           `json:"success_count"`   // 成功次数
	FailureCount   int           `json:"failure_count"`   // 失败次数
	SuccessRate    float64       `json:"success_rate"`    // 成功率
	AvgDuration    float64       `json:"avg_duration"`    // 平均执行时长（秒）
	MinDuration    float64       `json:"min_duration"`    // 最短执行时长（秒）
	MaxDuration    float64       `json:"max_duration"`    // 最长执行时长（秒）
	LastExecution  *TaskMetadata `json:"last_execution"`  // 最近一次执行
	FirstExecution *TaskMetadata `json:"first_execution"` // 最早一次执行
}

// GetStatistics 获取任务统计信息
func GetStatistics(logDir, taskName string, hours, days int) (*TaskStatistics, error) {
	executions, err := ListExecutions(logDir, taskName, hours, days)
	if err != nil {
		return nil, err
	}

	if len(executions) == 0 {
		return nil, fmt.Errorf("未找到任务 '%s' 的执行记录", taskName)
	}

	stats := &TaskStatistics{
		TaskName:       taskName,
		TotalCount:     len(executions),
		LastExecution:  executions[0],
		FirstExecution: executions[len(executions)-1],
	}

	var totalDuration float64
	stats.MinDuration = executions[0].Duration
	stats.MaxDuration = executions[0].Duration

	for _, exec := range executions {
		if exec.Status == "success" {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}

		totalDuration += exec.Duration

		if exec.Duration < stats.MinDuration {
			stats.MinDuration = exec.Duration
		}
		if exec.Duration > stats.MaxDuration {
			stats.MaxDuration = exec.Duration
		}
	}

	stats.AvgDuration = totalDuration / float64(stats.TotalCount)
	if stats.TotalCount > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
	}

	return stats, nil
}
