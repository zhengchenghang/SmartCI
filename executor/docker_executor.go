package executor

import (
    "context"
    "fmt"
    "io"
    "lite-cicd/config"
    "lite-cicd/core"
    "lite-cicd/metrics"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing"
)

type DockerExecutor struct {
    cli     *client.Client
    logDir  string
    imgPref string
}

func NewDockerExecutor(logDir string) (*DockerExecutor, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        return nil, err
    }
    return &DockerExecutor{cli: cli, logDir: logDir, imgPref: "smart-ci-"}, nil
}

func (e *DockerExecutor) Run(ctx context.Context, repo config.RepoConfig, branch string) (*core.TaskResult, error) {
    // 生成任务ID
    taskID := core.GenerateTaskID()
    
    // 创建任务目录
    taskDir, err := core.CreateTaskDir(e.logDir, taskID)
    if err != nil {
        return nil, fmt.Errorf("创建任务目录失败: %v", err)
    }
    
    // 生成日志文件路径（在任务目录中）
    logFile := filepath.Join(taskDir, "task.log")
    
    result := &core.TaskResult{
        TaskID:  taskID,
        TaskDir: taskDir,
        LogFile: logFile,
    }
    
    // 创建元数据记录
    metadata := &metrics.TaskMetadata{
        TaskID:    taskID,
        TaskName:  repo.Name,
        TaskType:  "repo",
        StartTime: time.Now(),
        LogFile:   logFile,
        TaskDir:   taskDir,
        Config: map[string]interface{}{
            "url":        repo.URL,
            "branch":     branch,
            "dockerfile": repo.Dockerfile,
            "test_cmd":   repo.TestCmd,
        },
    }
    
    log.Printf("🐳 [Docker] 任务ID: %s", taskID)
    log.Printf("📁 [Docker] 任务目录: %s", taskDir)
    
    workDir := filepath.Join("/tmp", "smart-ci", repo.Name, branch)

    // 1. Git Pull/Clone
    log.Printf("📥 [Git] 拉取代码: %s (%s)", repo.Name, branch)
    if err := e.syncCode(repo.URL, branch, workDir); err != nil {
        result.Error = fmt.Errorf("git sync failed: %v", err)
        metadata.EndTime = time.Now()
        metadata.Duration = metadata.EndTime.Sub(metadata.StartTime).Seconds()
        metadata.Status = "failure"
        metadata.Error = result.Error.Error()
        metrics.SaveMetadata(metadata)
        return result, result.Error
    }

    // 2. Docker Build
    tag := fmt.Sprintf("%s%s:%s", e.imgPref, strings.ToLower(repo.Name), branch)
    log.Printf("🐳 [Docker] 构建镜像: %s", tag)
    if err := e.buildImage(workDir, repo.Dockerfile, tag); err != nil {
        result.Error = fmt.Errorf("build failed: %v", err)
        metadata.EndTime = time.Now()
        metadata.Duration = metadata.EndTime.Sub(metadata.StartTime).Seconds()
        metadata.Status = "failure"
        metadata.Error = result.Error.Error()
        metrics.SaveMetadata(metadata)
        return result, result.Error
    }

    // 3. Run Test
    log.Printf("🚀 [Test] 运行测试...")
    err = e.runContainer(ctx, tag, repo.TestCmd, logFile)
    
    // 更新元数据
    metadata.EndTime = time.Now()
    metadata.Duration = metadata.EndTime.Sub(metadata.StartTime).Seconds()
    
    if err != nil {
        result.Error = err
        metadata.Status = "failure"
        metadata.Error = result.Error.Error()
    } else {
        metadata.Status = "success"
    }
    
    metrics.SaveMetadata(metadata)

    return result, err
}

// (Git 和 Docker 的底层实现与之前类似，为节省篇幅省略细节，重点在架构)
func (e *DockerExecutor) syncCode(url, branch, path string) error {
    // 简单实现：存在则 pull，不存在则 clone
    if _, err := os.Stat(path); os.IsNotExist(err) {
        _, err := git.PlainClone(path, false, &git.CloneOptions{
            URL: url, ReferenceName: plumbing.NewBranchReferenceName(branch), Depth: 1,
        })
        return err
    }
    r, _ := git.PlainOpen(path)
    w, _ := r.Worktree()
    return w.Pull(&git.PullOptions{ReferenceName: plumbing.NewBranchReferenceName(branch), Force: true})
}

func (e *DockerExecutor) buildImage(path, dockerfile, tag string) error {
    cmd := exec.Command("docker", "build", "-t", tag, "-f", filepath.Join(path, dockerfile), path)
    return cmd.Run() // 生产环境应捕获输出
}

func (e *DockerExecutor) runContainer(ctx context.Context, image, cmd, logPath string) error {
    // 创建并启动容器，将日志写入 logPath
    // 这里模拟运行过程
    resp, err := e.cli.ContainerCreate(ctx, &container.Config{
        Image: image, Cmd: []string{"sh", "-c", cmd + " > /test.log 2>&1"},
    }, nil, nil, nil, "")
    if err != nil {
        return err
    }

    defer e.cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
    if err := e.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
        return err
    }

    statusCh, errCh := e.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
    select {
    case err := <-errCh:
        return err
    case <-statusCh:
    }

    // 复制日志 (简化版)
    out, _, err := e.cli.CopyFromContainer(ctx, resp.ID, "/test.log")
    if err != nil {
        return err
    }
    defer out.Close()

    // 解压 tar 流并在本地保存 (省略 tar 解压代码，直接写入文件演示)
    f, _ := os.Create(logPath)
    defer f.Close()
    io.Copy(f, out)
    return nil
}
