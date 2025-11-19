package executor

import (
	"context"
	"fmt"
	"io"
	"lite-cicd/config"
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

func (e *DockerExecutor) Run(ctx context.Context, repo config.RepoConfig, branch string) (string, error) {
	workDir := filepath.Join("/tmp", "smart-ci", repo.Name, branch)

	// 1. Git Pull/Clone
	log.Printf("📥 [Git] 拉取代码: %s (%s)", repo.Name, branch)
	if err := e.syncCode(repo.URL, branch, workDir); err != nil {
		return "", fmt.Errorf("git sync failed: %v", err)
	}

	// 2. Docker Build
	tag := fmt.Sprintf("%s%s:%s", e.imgPref, strings.ToLower(repo.Name), branch)
	log.Printf("🐳 [Docker] 构建镜像: %s", tag)
	if err := e.buildImage(workDir, repo.Dockerfile, tag); err != nil {
		return "", fmt.Errorf("build failed: %v", err)
	}

	// 3. Run Test
	log.Printf("🚀 [Test] 运行测试...")
	logFile := filepath.Join(e.logDir, fmt.Sprintf("%s-%s-%d.log", repo.Name, branch, time.Now().Unix()))
	err := e.runContainer(ctx, tag, repo.TestCmd, logFile)

	return logFile, err
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
