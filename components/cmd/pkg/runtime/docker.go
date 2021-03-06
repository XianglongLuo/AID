// Copyright (c) 2020 Xiaozhe Yao & AICAMP.CO.,LTD
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/autoai-org/aid/components/cmd/pkg/entities"
	"github.com/autoai-org/aid/components/cmd/pkg/utilities"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/go-connections/nat"
	"github.com/logrusorgru/aurora"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var logger = utilities.NewDefaultLogger()
var defaultDockerRuntime *DockerRuntime

// DockerRuntime is the basic class for manage docker
type DockerRuntime struct {
	client *client.Client
}

// prepareEnvs returns a list of string
func prepareEnvs(imageID string) []string {
	reqPackage := entities.GetPackageByImageID(imageID)
	envs := entities.GetEnvironmentVariablesbyPackageID(reqPackage.ID, "all")
	envStrings := entities.MergeEnvironmentVariables(envs)
	serverIP := utilities.GetOutboundIP().String()
	systemenv := []string{"AID_SERVER", serverIP + ":10590"}
	envStrings = append(envStrings, systemenv...)
	return envStrings
}

// prepareVolumes returns a map of volume mappings
func prepareVolumes(hostPath string) map[string]struct{} {
	vol := map[string]struct{}{hostPath + ":/package": {}}
	return vol
}

// NewDockerRuntime returns a DockerRuntime Instance
func NewDockerRuntime() *DockerRuntime {
	if defaultDockerRuntime != nil {
		return defaultDockerRuntime
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		utilities.CheckError(err, "Cannot Create New Docker Runtime")
	}
	return &DockerRuntime{
		client: cli,
	}
}

// Pull will pull an exisiting package
func (docker *DockerRuntime) Pull(imageName string) error {
	reader, err := docker.client.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
	if err != nil {
		utilities.CheckError(err, "Cannot pull image "+imageName)
		return err
	}
	io.Copy(os.Stdout, reader)
	return nil
}

// Create will create a container from an image
func (docker *DockerRuntime) Create(imageID string, hostPort string) (container.ContainerCreateCreatedBody, error) {
	image := entities.GetImage(imageID)
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			},
		},
	}
	resp, err := docker.client.ContainerCreate(context.Background(), &container.Config{
		Image: image.Name,
		Tty:   true,
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
		Env: prepareEnvs(image.ID),
	}, hostConfig, nil, "")
	if err != nil {
		utilities.CheckError(err, "Cannot create container from image "+image.Name)
		return resp, err
	}
	container := entities.Container{ID: resp.ID, ImageID: imageID}
	container.Save()
	runningSolver := entities.RunningSolver{ContainerID: resp.ID, Status: "Ready", EntryPoint: "127.0.0.1:" + hostPort}
	runningSolver.Save()
	return resp, err
}

// Start will start a container
func (docker *DockerRuntime) Start(containerID string) error {
	container := entities.GetContainer(containerID)
	runningSolvers := entities.RunningSolver{ContainerID: container.ID, Status: "Running"}
	runningSolvers.Save()
	if err := docker.client.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); err != nil {
		utilities.CheckError(err, "Cannot start container "+containerID)
		return err
	}
	return nil
}

func getBuildCtx(solverPath string) io.Reader {
	ctx, _ := archive.TarWithOptions(solverPath, &archive.TarOptions{})
	return ctx
}

func realBuild(docker *DockerRuntime, dockerfile string, imageName string, buildLogger *logrus.Logger) error {
	if !utilities.IsFileExists(dockerfile) {
		utilities.Formatter.Warning("Dockerfile Not Found, AID will generate it for you.")
		GenerateDockerFiles(filepath.Dir(dockerfile))
		tomlFilePath := filepath.Join(filepath.Dir(dockerfile), "aid.toml")
		aidToml, err := utilities.ReadFileContent(tomlFilePath)
		if err != nil {
			utilities.CheckError(err, "Cannot open file "+tomlFilePath)
		}
		solvers := entities.LoadSolversFromConfig(aidToml)
		RenderRunnerTpl(filepath.Dir(dockerfile), solvers)
	}
	buildResponse, err := docker.client.ImageBuild(context.Background(), getBuildCtx(path.Dir(dockerfile)), types.ImageBuildOptions{
		Tags:       []string{imageName},
		Dockerfile: filepath.Base(dockerfile),
		Remove:     true,
	})
	if err != nil {
		utilities.CheckError(err, "Cannot build image "+imageName)
		buildLogger.Error("Cannot build image " + imageName)
		buildLogger.Error(err.Error())
	}
	// set logs to build logs
	reader := buildResponse.Body
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		var buildLog entities.BuildLog
		json.Unmarshal(scanner.Bytes(), &buildLog)
		buildLogger.Info(buildLog.Stream)
	}
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	err = jsonmessage.DisplayJSONMessagesStream(buildResponse.Body, os.Stderr, termFd, isTerm, nil)
	if err != nil {
		buildLogger.Error("Cannot show log from " + imageName)
		buildLogger.Error(err.Error())
	}
	return err
}

// Build will build a new image from dockerfile
func (docker *DockerRuntime) Build(imageName string, dockerfile string) (entities.Log, error) {
	var err error
	nextCount := entities.GetNextImageNumber()
	fullImageName := "aid-" + nextCount + "-" + imageName
	logger.Info("Starting Build Image: " + fullImageName + "...")
	logid := utilities.GenerateUUIDv4()
	var logPath = filepath.Join(utilities.GetBasePath(), "logs", "builds", imageName+"."+logid)
	log := entities.NewLogObject("build-"+fullImageName, logPath, "docker-build")
	log.ID = logid
	log.Save()
	buildLogger := utilities.NewLogger(logPath)
	err = realBuild(docker, dockerfile, fullImageName, buildLogger)
	if err == nil {
		// no error occured, add image to database
		image := entities.Image{Name: fullImageName}
		image.Save()
		ibe := entities.ImageBuiltEvent{ImageName: fullImageName}
		entities.Consume(ibe)
	}
	return log, err
}

// ListImages returns all images that have been installed on the host
func (docker *DockerRuntime) ListImages() []types.ImageSummary {
	images, err := docker.client.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		logger.Error("Cannot List Images")
		logger.Error(err.Error())
	}
	return images
}

// ListContainers returns all containers that have been built.
func (docker *DockerRuntime) ListContainers() []types.Container {
	containers, err := docker.client.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		logger.Error("Cannot List Images")
		logger.Error(err.Error())
	}
	return containers
}

// GenerateDockerFiles returns a DockerFile string that could be used to build image.
func GenerateDockerFiles(baseFilePath string) {
	configFile := filepath.Join(baseFilePath, "aid.toml")
	tomlString, err := utilities.ReadFileContent(configFile)
	if err != nil {
		utilities.CheckError(err, "cannot open file "+configFile)
	}
	packageInfo := entities.LoadPackageFromConfig(tomlString)
	for _, solver := range packageInfo.Solvers {
		RenderDockerfile(solver.Name, baseFilePath)
	}
}

// FetchContainerLogs returns the logs in the container
func (docker *DockerRuntime) FetchContainerLogs(containerID string) entities.Log {
	var logPath = filepath.Join(utilities.GetBasePath(), "logs", "container", containerID)
	log := entities.NewLogObject("container-"+containerID, logPath, "container-log")
	log.Save()
	logger := utilities.NewLogger(logPath)
	reader, err := docker.client.ContainerLogs(context.Background(), containerID, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
		Follow:     true,
		Tail:       "40",
	})
	if err != nil {
		utilities.CheckError(err, "Cannot establish error log reader")
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logger.Info(scanner.Text())
	}
	return log
}

// ExportImage save image into dest, for uploading to other nodes, etc.
// Before calling export, we should ask users if they want to overwrite existing file.
func (docker *DockerRuntime) ExportImage(imageName string) error {
	targetFile := filepath.Join(utilities.GetBasePath(), "temp", imageName+".aidimg")
	fmt.Printf("%s Exporting your image to %s\n", aurora.Cyan("progress"), targetFile)
	// Delete existing file
	if utilities.IsExists(targetFile) {
		err := os.Remove(targetFile)
		utilities.CheckError(err, "Cannot remove existing file "+targetFile)
	}
	file, err := os.Create(targetFile)
	if err != nil {
		utilities.CheckError(err, "Cannot create new image file at "+targetFile)
		return err
	}
	defer file.Close()
	resBody, err := docker.client.ImageSave(context.Background(), []string{imageName})
	_, err = io.Copy(file, resBody)
	if err != nil {
		utilities.CheckError(err, "Cannot write to the file: "+targetFile)
		return err
	}
	utilities.Formatter.Success("exported your image to " + targetFile)
	return nil
}

// ImportImage reads images into docker runtime
func (docker *DockerRuntime) ImportImage(imageName string, quiet bool) error {
	targetFile := filepath.Join(utilities.GetBasePath(), "temp", imageName+".aidimg")
	fmt.Printf("%s Importing your image from %s\n", aurora.Cyan("progress"), targetFile)
	_, err := os.Stat(targetFile)
	if os.IsNotExist(err) {
		utilities.CheckError(err, "Cannot read the file: "+targetFile)
		return err
	}
	file, err := os.Open(targetFile)
	if err != nil {
		utilities.CheckError(err, "Cannot open the file: "+targetFile)
	}
	reader := bufio.NewReader(file)
	_, err = docker.client.ImageLoad(context.Background(), reader, quiet)
	if err != nil {
		utilities.CheckError(err, "Cannot import image from "+targetFile)
		return err
	}
	fmt.Printf("imported your image from " + targetFile)
	return nil
}
