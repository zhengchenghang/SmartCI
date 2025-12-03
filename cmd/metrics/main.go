package main

import (
	"flag"
	"fmt"
	"lite-cicd/metrics"
	"os"
)

func main() {
	// 定义子命令
	latestCmd := flag.NewFlagSet("latest", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	statsCmd := flag.NewFlagSet("stats", flag.ExitOnError)
	allCmd := flag.NewFlagSet("all", flag.ExitOnError)

	// latest 子命令参数
	latestTask := latestCmd.String("task", "", "任务名称 (必需)")
	latestLogDir := latestCmd.String("logdir", "./logs", "日志目录")

	// list 子命令参数
	listTask := listCmd.String("task", "", "任务名称 (必需)")
	listLogDir := listCmd.String("logdir", "./logs", "日志目录")
	listHours := listCmd.Int("hours", 0, "最近多少小时")
	listDays := listCmd.Int("days", 0, "最近多少天")
	listLimit := listCmd.Int("limit", 20, "最多显示条数")

	// stats 子命令参数
	statsTask := statsCmd.String("task", "", "任务名称 (必需)")
	statsLogDir := statsCmd.String("logdir", "./logs", "日志目录")
	statsHours := statsCmd.Int("hours", 0, "最近多少小时")
	statsDays := statsCmd.Int("days", 0, "最近多少天")

	// all 子命令参数
	allLogDir := allCmd.String("logdir", "./logs", "日志目录")

	// 检查参数
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// 根据子命令执行相应操作
	switch os.Args[1] {
	case "latest":
		latestCmd.Parse(os.Args[2:])
		if *latestTask == "" {
			fmt.Println("❌ 错误: 必须指定任务名称")
			latestCmd.Usage()
			os.Exit(1)
		}
		handleLatest(*latestLogDir, *latestTask)

	case "list":
		listCmd.Parse(os.Args[2:])
		if *listTask == "" {
			fmt.Println("❌ 错误: 必须指定任务名称")
			listCmd.Usage()
			os.Exit(1)
		}
		handleList(*listLogDir, *listTask, *listHours, *listDays, *listLimit)

	case "stats":
		statsCmd.Parse(os.Args[2:])
		if *statsTask == "" {
			fmt.Println("❌ 错误: 必须指定任务名称")
			statsCmd.Usage()
			os.Exit(1)
		}
		handleStats(*statsLogDir, *statsTask, *statsHours, *statsDays)

	case "all":
		allCmd.Parse(os.Args[2:])
		handleAll(*allLogDir)

	default:
		fmt.Printf("❌ 未知子命令: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("SmartCI 任务统计工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  metrics <command> [options]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  latest   显示指定任务的最近一次执行记录")
	fmt.Println("  list     列出指定任务的历史执行记录")
	fmt.Println("  stats    显示指定任务的统计信息")
	fmt.Println("  all      显示所有任务的简要统计")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  metrics latest -task backup-database")
	fmt.Println("  metrics list -task backup-database -days 7")
	fmt.Println("  metrics stats -task backup-database -days 30")
	fmt.Println("  metrics all")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -task string     任务名称 (latest/list/stats 必需)")
	fmt.Println("  -logdir string   日志目录 (默认: ./logs)")
	fmt.Println("  -hours int       最近多少小时 (list/stats 可选)")
	fmt.Println("  -days int        最近多少天 (list/stats 可选)")
	fmt.Println("  -limit int       最多显示条数 (list, 默认: 20)")
}

func handleLatest(logDir, taskName string) {
	metadata, err := metrics.GetLatestExecution(logDir, taskName)
	if err != nil {
		fmt.Printf("❌ 获取最近执行记录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(metrics.DisplayLatestExecution(metadata))
}

func handleList(logDir, taskName string, hours, days, limit int) {
	executions, err := metrics.ListExecutions(logDir, taskName, hours, days)
	if err != nil {
		fmt.Printf("❌ 获取执行记录失败: %v\n", err)
		os.Exit(1)
	}

	if len(executions) == 0 {
		fmt.Printf("📭 任务 '%s' 没有找到执行记录\n", taskName)
		return
	}

	// 限制显示条数
	if len(executions) > limit {
		executions = executions[:limit]
	}

	fmt.Println(metrics.DisplayExecutionList(executions, taskName))
}

func handleStats(logDir, taskName string, hours, days int) {
	stats, err := metrics.GetStatistics(logDir, taskName, hours, days)
	if err != nil {
		fmt.Printf("❌ 获取统计信息失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(metrics.DisplayStatistics(stats, hours, days))
}

func handleAll(logDir string) {
	allMetadata, err := metrics.ListAllMetadata(logDir)
	if err != nil {
		fmt.Printf("❌ 获取任务列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(allMetadata) == 0 {
		fmt.Println("📭 没有找到任何执行记录")
		return
	}

	fmt.Println(metrics.DisplayAllTasksSummary(allMetadata))
}
