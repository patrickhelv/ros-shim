package connection

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	pb "ros2shim/protoexport/service"
	"ros2shim/utility"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"
)

type childPidsValues struct {
	pid  int32
	timestamp string
	time time.Time
}

type task struct {
	done    bool
	message string
}

type LogServiceServer struct {
	mu          sync.Mutex
	ChildPids   map[string]childPidsValues
	Time        time.Time
	RosNodeinfo *RosNodeInfo
	tasks       map[string]task
	connectiondata  map[string]bool
	KeyPath string
	pb.UnimplementedRosShimServiceServer
	quit chan os.Signal
}

// generates a new task Id
func generateNewTaskId() string {
	unixTime := time.Now().Unix()

	s, _ := rand.Int(rand.Reader, big.NewInt(100))
	t, _ := rand.Int(rand.Reader, big.NewInt(1000))

	v, _ := rand.Int(rand.Reader, big.NewInt(10))

	timebytes := []byte(strconv.Itoa(int(unixTime) - int(s.Int64()) + int(t.Uint64()) + int(2*v.Int64())))

	hasher := sha256.New()
	hasher.Write(timebytes)
	hashBytes := hasher.Sum(nil)

	hashString := hex.EncodeToString(hashBytes)

	shortHashString := hashString[:10]

	return shortHashString
}

func (s *LogServiceServer) SetConnectionData(ip string, status bool){
	s.connectiondata[ip] = status
}

// function to updated the rosNodeInfo that holds most of the project files, and launcher files used
// to start a project
func (s *LogServiceServer) UpdateRosNodeInfo(rosNodeInfo *RosNodeInfo) {
	
	log.Println("update ros node info called")
	s.RosNodeinfo = rosNodeInfo
	s.Time = time.Now()
	s.connectiondata = make(map[string]bool)
	s.ChildPids = make(map[string]childPidsValues)
}

// updateTaskStatus safely updates the status of a given task.
func (s *LogServiceServer) updateTaskStatus(taskID string, isCompleted bool, result string) {
	
	log.Println("update Task Status called")
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		task.done = isCompleted
		task.message = result
		s.tasks[taskID] = task // Update the task in the map
	}
}

// Return the list of child PIDs associated with the process
func (s *LogServiceServer) GetChildPids(ctx context.Context, req *pb.Empty) (*pb.ChildPidsResponse, error) {

	log.Println("Get child pids called")
	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}

	protoMessages := make([]*pb.ChildPids, 0)

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.ChildPids {
		protoMessages = append(protoMessages, &pb.ChildPids{
			Pid: v.pid,
		})
	}
	return &pb.ChildPidsResponse{Pids: protoMessages}, nil
}

// Return the number of child PIDs associated with the process
func (s *LogServiceServer) GetAmountOfChilds(ctx context.Context, req *pb.Empty) (*pb.ChildAmount, error) {
	
	log.Println("Get amount of childs called")
	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()

	length := int32(len(s.ChildPids))
	return &pb.ChildAmount{Pids: length}, nil
}

// CheckTaskStatus returns the status of a task by its ID.
func (s *LogServiceServer) CheckTaskStatus(ctx context.Context, req *pb.TaskStatus) (*pb.StatusResponse, error) {
	log.Println("Check task status called")

	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[req.TaskId]

	if !exists {
		log.Println("Task not found")
		return &pb.StatusResponse{
			Success: false,
			Message: "Task not found",
		}, nil
	}

	if task.done {

		return &pb.StatusResponse{
			Success: task.done,
			Message: task.message,
		}, nil
	}

	return &pb.StatusResponse{
		Success: task.done,
		Message: "Waiting...",
	}, nil

}


// retrieves the child pid that is associated to the launch file sent in as an argument
// returns (LogInfo / nil)
// LogInfo contains the launchfile and the child pid (string)
func (s *LogServiceServer) GetLogs(ctx context.Context, req *pb.LaunchFile) (*pb.LogInfo, error) {
	var data []string
	log.Println("Get logs service called")

	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}

	launchFile := req.GetLaunchFileName()

	if val, exists := s.ChildPids[launchFile]; exists {
		
		logFilePath, err := utility.FindLogFile(s.RosNodeinfo.LogPath, strconv.Itoa(int(val.pid+1)), val.timestamp)
		
		fmt.Println(logFilePath)
		if err != nil{
			log.Printf("there is an error finding the log file, with pid %d and logpath %s, with error %v", val.pid, s.RosNodeinfo.LogPath, err)
			return nil, fmt.Errorf("the log file was not found")
		}

		launchpid, err := utility.ExtractPIDFromLog(logFilePath)

		if err != nil {
			log.Printf("We could not extract the pid from the log file, %s, with error %v", logFilePath, err)
			return nil, fmt.Errorf("error finding logs")
		}

		logFileName, err := utility.GetLogFilename(launchpid, s.RosNodeinfo.LogPath)

		if err != nil {
			log.Printf("The log file does not exist for launch file %s", launchFile)
			return nil, fmt.Errorf("error finding logs")
		}

		log.Printf("log path : %s\n", s.RosNodeinfo.LogPath + "/" + logFileName)

		logLaunch := utility.ReadFromLog(s.RosNodeinfo.LogPath + "/" + logFileName)

		data = append(data, ".") // color formatting
		data = append(data, "Log from launch file")
		data = append(data, ".")
		data = append(data, "Showing the last 10 log lines..")
		data = append(data, "\n")

		data = append(data, logLaunch...)

		data = append(data, "\n\n")
		data = append(data, ".")
		data = append(data, fmt.Sprintf("log launching %s", req.LaunchFileName))

		logPreLaunch := utility.ReadFromLog(logFilePath)

		data = append(data, logPreLaunch...)
		
		logResponse := &pb.LogInfo{
			Launchfile: launchFile,
			Message: data,
		}

		return logResponse, nil

	}else{

		return nil, fmt.Errorf("did not find launchfile")

	}

}

// Stops the ros nodes associated with the launch files
// returns a list of (bool / message) ordered on the list of launchfiles sent
// the bool represents if the operation was successfull
// the message contains the launchfile
// returns either true with the associated launchfile to indicate successfull ros node shutdown
// returns false with the associated launchfile to indicate unsuccessfull ros node shutdown
// returns false with "-" + the name of the associated launch to indicate the launchfile does not exist
func (s *LogServiceServer) StopRosNode(ctx context.Context, req *pb.LaunchFile) (*pb.StatusResponse, error) {
	
	ip := findIpFromPeerConnection(ctx)
	log.Println("Stop service called")

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, fmt.Errorf("node not registered")
	}

	launchFile := req.GetLaunchFileName()

	if val, exists := s.ChildPids[launchFile]; exists { // if exists then shutdownchild

		log.Printf("Shutting down ros node with pid %d\n", val.pid)
		if ShutDownChild(int(val.pid)) {
			statusResponse := &pb.StatusResponse{ //store the status response
				Success: true,
				Message: launchFile,
			}

			delete(s.ChildPids, launchFile) // delete the child pid with the associated launch file
			return statusResponse, nil

		} else { // if shutdown does not succeed
			statusResponse := &pb.StatusResponse{
				Success: false,
				Message: launchFile,
			}
			return statusResponse, fmt.Errorf("shutdown did not succeed")
		}

	} else { // if launchfile does not exist
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: launchFile,
		}
		return statusResponse, fmt.Errorf("ros project was not found on this node")

	}

	// sends back all the status responses with bool ans message to indicate the success
}

// Starts the ros nodes associated with the launch files
// Returns the StatusResponse with value True and Taskid
// the bool represents if the operation was successfull
// the message contains the taskID
// This enables us to no longer busy wait for the ros application to 
// start, we update the status of the task that is sent to the controller.

// If the service fails we update the task with the correct error message.
func (s *LogServiceServer) StartService(ctx context.Context, req *pb.StartParams) (*pb.StatusResponse, error) {

	log.Println("Start service called")
	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, fmt.Errorf("node not registered")
	}

	taskId := generateNewTaskId()
	s.mu.Lock()
	s.tasks[taskId] = task{
		done:    false,
		message: "starting",
	}
	s.mu.Unlock()

	go func(s *LogServiceServer, taskId string, req *pb.StartParams) {

		launchFile := req.LaunchFileName

		workdir := s.RosNodeinfo.WorkpackageDirectory + "/" + req.Workpackage

		status,_ := utility.VerifyProjectExistence(workdir, req.LaunchFileName)
		isprojectBuild := true

		if !status {
			if !CloneProject(req.GitUrl, req.GitBranch, workdir) {
				s.updateTaskStatus(taskId, true, fmt.Sprintf("Failed to clone the repository %s on branch %s", req.GitUrl, req.GitBranch))
				return
			}
		} else {
			status := UpdateGitRepository(workdir)
			log.Printf("updated repository status : %v\n", status)
		}

		if req.VenvRequired{
			if !InitVenvAndRequirements(s.RosNodeinfo, workdir){
				s.updateTaskStatus(taskId, true, fmt.Sprintf("Failed to create a new venv and build requirements.txt for the project %s", req.GetRosProjectName()))
				return
			}
		}
		
		if !utility.IsProjectBuilt(workdir) {
			isprojectBuild = false
			if !BuildProject(*s.RosNodeinfo, workdir, req.GetRosProjectName(), req.VenvRequired) {
				s.updateTaskStatus(taskId, true, fmt.Sprintf("Failed to build the ros project %s with launch file %s", req.GetRosProjectName(), req.GetLaunchFileName()))
				return
			}
		}

		if !SourceProject(*s.RosNodeinfo, workdir, req.VenvRequired) {
			
			if isprojectBuild{
				if !BuildProject(*s.RosNodeinfo, workdir, req.GetRosProjectName(), req.VenvRequired) {
					s.updateTaskStatus(taskId, true, fmt.Sprintf("Failed to build the ros project %s with launch file %s", req.GetRosProjectName(), req.GetLaunchFileName()))
					return
				}
			}
			if !SourceProject(*s.RosNodeinfo, workdir, req.VenvRequired) {
				s.updateTaskStatus(taskId, true, fmt.Sprintf("Failed to source the new ros project %s", req.GetRosProjectName()))
				return
			}
		}

		if exists := utility.ExistsIn(launchFile, s.RosNodeinfo.LaunchFiles); !exists {
			s.RosNodeinfo.LaunchFiles = append(s.RosNodeinfo.LaunchFiles, launchFile)
		}

		val, err := StartLaunch(*s.RosNodeinfo, launchFile, req.GetRosProjectName(), workdir, req.VenvRequired)
		timeNow := time.Now()
		timestamp := timeNow.Format("2006-01-02-15")

		log.Printf("Child pid: %d for the launch file : %s", val, launchFile)

		if err != nil {	
			s.updateTaskStatus(taskId, false, "The ros node failed to start")

		} else {

			s.ChildPids[launchFile] = childPidsValues{pid: int32(val), time: timeNow, timestamp: timestamp}
			log.Printf("childPids list %v", s.ChildPids)
			s.updateTaskStatus(taskId, true, "The ros node started successfully")
		}


	}(s, taskId, req)

	statusResponse := &pb.StatusResponse{
		Success: true,
		Message: taskId,
	}

	return statusResponse, nil

}

// Restarts the ros nodes associated with the launch files
// returns a list of (bool / message) ordered on the list of launchfiles sent
// the bool represents if the operation was successfull
// the message contains the launchfile
// returns either true with the associated launchfile to indicate successfull ros node start
// returns false with the associated launchfile to indicate unsuccessfull ros node start
// returns false with "-" + the name of the associated launch to indicate the launchfile does not exist
func (s *LogServiceServer) RestartService(ctx context.Context, req *pb.StartParams) (*pb.StatusResponse, error) {

	log.Println("restart service called")
	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, fmt.Errorf("node not registered")
	}

	statusResponseStop, err := s.StopRosNode(ctx, &pb.LaunchFile{LaunchFileName: req.GetLaunchFileName()})
	if err != nil{
		return nil, fmt.Errorf("stop ros node failed with error %v", err)
	}

	if statusResponseStop.Success {
		statusResponseStart, err := s.StartService(ctx, req) // if start fail return stop

		if err != nil{
			return nil, fmt.Errorf("start launching ros node %s", req.LaunchFileName)
		}

		statusResponse := &pb.StatusResponse{
			Success: true,
			Message: statusResponseStart.Message,
		}

		return statusResponse, nil
	}

	statusResponse := &pb.StatusResponse{
		Success: false,
		Message: fmt.Sprintf("Failed to Stop the launch file %s", req.LaunchFileName),
	}

	return statusResponse, err
} 

// Checks the Health of Node controller
// returns (bool/int) the bool represents that the node is healthy
// the int represents the uptime in seconds of the RosNode
func (s *LogServiceServer) CheckHealth(ctx context.Context, req *pb.Empty) (*pb.HealthResponse, error) {

	ip := findIpFromPeerConnection(ctx)
	log.Println("Checking health node ...")

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}

	uptime := time.Since(s.Time)
	uptimeSeconds := int64(uptime.Seconds())

	return &pb.HealthResponse{Healthy: true, UptimeSeconds: uptimeSeconds}, nil
}

// Checks the Health of the ros nodes
// returns (bool/string/int) the bool represents that the node is healthy
// the string represents the launch file associated with this node
// the int represents the uptime in seconds of the RosNode
// We return -2 as the uptime if the launch file does not exist
// we return -1 as the process is no longer active
func (s *LogServiceServer) CheckHealthRosNodes(ctx context.Context, req *pb.Empty) (*pb.HealthResponses, error) {

	log.Println("check health of all ros nodes called")

	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}

	var statusResponsesList []*pb.HealthResponseRosNode
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("CheckHealthRosNode has been called")

	for launchFile := range s.ChildPids {
		log.Printf("child pid in the list: %d", s.ChildPids[launchFile].pid)

		if val, exists := s.ChildPids[launchFile]; exists {

			log.Printf("child pid : %d", val.pid)
			if utility.IsProcessActive(int(val.pid)) {
				uptime := time.Since(val.time)

				statusResponse := &pb.HealthResponseRosNode{
					Healthy:       true,
					Launchfile:    launchFile,
					UptimeSeconds: int64(uptime.Seconds()),
				}
				statusResponsesList = append(statusResponsesList, statusResponse)
			} else { // process is not active
				statusResponse := &pb.HealthResponseRosNode{
					Healthy:       false,
					Launchfile:    launchFile,
					UptimeSeconds: -1,
				}
				statusResponsesList = append(statusResponsesList, statusResponse)
			}
		} else { // if launch file does not exist
			statusResponse := &pb.HealthResponseRosNode{
				Healthy:       false,
				Launchfile:    launchFile,
				UptimeSeconds: -2,
			}
			statusResponsesList = append(statusResponsesList, statusResponse)
		}
	}

	statusResponses := &pb.HealthResponses{
		HealthResponseRosNode: statusResponsesList,
	}

	return statusResponses, nil

}

// Checks the Health of the ros nodes
// returns (bool/string/int) the bool represents that the node is healthy
// the string represents the launch file associated with this node
// the int represents the uptime in seconds of the RosNode
// We return -2 as the uptime if the launch file does not exist
// we return -1 as the process is no longer active
func (s *LogServiceServer) CheckHealthRosNode(ctx context.Context, req *pb.LaunchFile) (*pb.HealthResponse, error) {

	log.Printf("check health of one node %s\n", req.LaunchFileName)

	ip := findIpFromPeerConnection(ctx)

	if !s.isConnectionValid(ip){
		log.Println("Node not registered")
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("CheckHealthRosNode has been called")

	log.Printf("child pid in the list: %d", s.ChildPids[req.GetLaunchFileName()].pid)

	if val, exists := s.ChildPids[req.GetLaunchFileName()]; exists {

		log.Printf("child pid : %d", val.pid)
		if utility.IsProcessActive(int(val.pid)) {
			uptime := time.Since(val.time)

			statusResponse := &pb.HealthResponse{
				Healthy:       true,
				UptimeSeconds: int64(uptime.Seconds()),
			}

			return statusResponse, nil

		} else { // process is not active
			statusResponse := &pb.HealthResponse{
				Healthy:       false,
				UptimeSeconds: -1,
			}

			return statusResponse, nil

		}
	} else { // if launch file does not exist
		statusResponse := &pb.HealthResponse{
			Healthy:       false,
			UptimeSeconds: -2,
		}
		return statusResponse, nil

	}

}

//TODO create DeRegister method

func (s *LogServiceServer) DeRegister(ctx context.Context, req *pb.Empty)(*pb.StatusResponse, error){
	statusResponse := &pb.StatusResponse{
		Success: true,
		Message: "TODO",
	}

	return statusResponse, nil
}

// Returns the ip address from the connection peer.
func findIpFromPeerConnection(ctx context.Context) (string){

	p, ok := peer.FromContext(ctx)
	if !ok {
        log.Println("Failed to get peer from context")
        return ""
    }
	// Extracting the IP address from the peer information
    
	if peerAddr, ok := p.Addr.(*net.TCPAddr); ok {
        clientIP := peerAddr.IP.String()
        log.Printf("Client IP address: %s", clientIP)
		return clientIP
    } else {
        log.Println("Failed to assert type *net.TCPAddr from peer.Addr")
		return ""
    }

}

func (s *LogServiceServer) isConnectionValid(ip string) (bool){
	// Check if the key exists in the map
    value, exists := s.connectiondata[ip]
    if exists {
        fmt.Printf("Key '%s' exists with value: %t\n", ip, value)
		return value
    } else {
        fmt.Printf("Key '%s' does not exist\n", ip)
		return false
    }
}

// Register the shim with the ros controller
func (s *LogServiceServer) Register(ctx context.Context, req *pb.Secret) (*pb.StatusResponse, error){

	log.Println("Register called..")

	ip := findIpFromPeerConnection(ctx)

	if ip == ""{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "failed to get connection information",
		}
		return statusResponse, fmt.Errorf("failed to get connection information")
	}

	if status := !utility.IsValidJWT(req.GetMessage()); !status{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "JWT not valid token",
		}
		return statusResponse, fmt.Errorf("jwt is not a valid token")
	}

	keydata := utility.ReadFile(s.KeyPath)
	key, err := utility.PemToECDSAPub(string(keydata))

	if err != nil{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "There was an error",
		}
		return statusResponse, fmt.Errorf("there was an error")
	}


	status, err := utility.CheckTokenExpiry(req.GetMessage(), key)

	if status && err != nil{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "The token is wrong",
		}
		return statusResponse, fmt.Errorf("the token is wrong")
	}

	s.SetConnectionData(ip, true)

	statusResponse := &pb.StatusResponse{
		Success: true,
		Message: fmt.Sprintf("power:%v", s.RosNodeinfo.power),
	}

	fmt.Printf("the connection with ip %s registered successfully\n", ip)

	return statusResponse, nil
}

// setup grpc communication between the manager and control
func (s *LogServiceServer) StartGrpcCommunication(port, serverCertPath, serverCertKey, messagePathKey, caPemPath string) {

	log.Println("Start grpc server called..")
	certPool := x509.NewCertPool()
	caPem := utility.ReadFile(caPemPath)
	if ok := certPool.AppendCertsFromPEM(caPem); !ok {
		log.Fatalf("failed to append ca certs\n")
	}

	s.KeyPath = messagePathKey
	listen, err := net.Listen("tcp", "0.0.0.0" + ":"+port)
	if err != nil {
		log.Printf("Failed to create a listener on port %s\n", port)
		return
	}

	certBytes := utility.ReadFile(serverCertPath)
	keyBytes := utility.ReadFile(serverCertKey)

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}

	tlsCredentials := credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: certPool})

	s.Time = time.Now()
	s.tasks = make(map[string]task)

	server := grpc.NewServer(grpc.Creds(tlsCredentials))

	// Create a health server instance
	healthServer := health.NewServer()

	s.quit = make(chan os.Signal, 1)

	go func() {
		pb.RegisterRosShimServiceServer(server, s)

		grpc_health_v1.RegisterHealthServer(server, healthServer)

		healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

		log.Printf("Server listening on %s\n", port)
		if err := server.Serve(listen); err != nil {
			log.Printf("Failed to serve: %v\n", err)
			return
		}
	}()

	// catch SIGINT (Ctrl+C) and SIGTERM ((Kill)
	signal.Notify(s.quit, syscall.SIGINT, syscall.SIGTERM)
	<-s.quit
	server.GracefulStop()
}


// Updates the children PID
func (s *LogServiceServer) UpdateChildPid(launchfile string, pid int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	startTime := time.Now()

	s.ChildPids["launchfile"] = childPidsValues{pid: int32(pid), time: startTime}
}

// gets the general CPU usage and memory usage for the host
func (s *LogServiceServer) GetSystemUtilization(ctx context.Context, req *pb.Empty) (*pb.DataUtilizationSystem, error){
	
	log.Println("get system utilization called")
	ip := findIpFromPeerConnection(ctx)

	if ip == ""{
		return nil, fmt.Errorf("failed to get connection information")
	}

	cpuUsage, memoryUsage, err := utility.GetGeneralUsage()

	if err != nil{
		return nil, fmt.Errorf("failed to retrieve information")
	}

	dataUtilization := &pb.DataUtilizationSystem{
		CpuPercent: cpuUsage,
		VmemPercent: memoryUsage,
	}

	fmt.Printf("cpu and memory data : %s and %s\n", dataUtilization.CpuPercent, dataUtilization.VmemPercent)

	return dataUtilization, nil
	
}

// Provides the CPU Percent, CPU usage and memory usage in percent for the ROS process
func (s *LogServiceServer) GetRosNodeUtilization(ctx context.Context, req *pb.LaunchFile) (*pb.DataUtilizationRosNode, error){
	
	log.Println("Get system utilization ros nodes called")
	
	ip := findIpFromPeerConnection(ctx)

	if ip == ""{
		return nil, fmt.Errorf("failed to get connection information")
	}
	log.Printf("childPids list %v", s.ChildPids)

	if val, exists := s.ChildPids[req.LaunchFileName]; exists {

		cpuPercent, cpuUsage, memUsagePercent, err := utility.GetProcessUsageFromPid(int(val.pid))

		log.Println(cpuPercent, cpuUsage)

		if err != nil{
			log.Printf("error could not get usage from pid %d with error %v\n", val.pid, err)
			return nil, fmt.Errorf("failed to retrieve information")
		}
	
		dataUtilization := &pb.DataUtilizationRosNode{
			CpuPercent: cpuPercent,
			CpuUsage: cpuUsage,
			MemUsagePercent: memUsagePercent,
		}

		return dataUtilization, nil
	}

	return nil, fmt.Errorf("the launch file does not exist on this node")
	
}

// DO NOT USE
func (s *LogServiceServer) RestartRos2Shim(ctx context.Context, req *pb.Secret) (*pb.StatusResponse, error){

	log.Println("restart shim called")

	ip := findIpFromPeerConnection(ctx)

	if ip == ""{
		return nil, fmt.Errorf("failed to get connection information")
	}

	if status := !utility.IsValidJWT(req.GetMessage()); !status{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "JWT not valid token",
		}
		return statusResponse, fmt.Errorf("jwt is not a valid token")
	}

	for val,entry := range s.ChildPids{
		if !utility.TerminateProcess(int(entry.pid)){
			log.Printf("There was an error terminating %s with pid %d", val, entry.pid)
		}
	}

	log.Println("Restarting ...")
	
	go func() {
		const maxRetries = 5

		home, err := os.UserHomeDir()

		if err != nil {
			log.Fatalf("Error getting home directory: %s\n", err)	
		}

		for i := 0; i < maxRetries; i++ {
			if i > 0 {
				log.Println("Attempting to restart again...")
				time.Sleep(2 * time.Second) // Increase the sleep time for each retry
			}
			cmd := exec.Command(home + "/ros2-node-shim/ros2-node-shim.bin")
			
			if err := cmd.Start(); err == nil {
				os.Exit(0) // Exit successfully if restart command succeeds
			}
			log.Printf("Restart attempt %d failed: %v", i+1, err)
		}
		// All retries failed, handle as needed
		log.Println("All restart attempts failed")
	}()

	status := pb.StatusResponse{
		Success: true,
		Message: "successfully restarted the ros node shim",
	}

	return &status, nil
}

func (s *LogServiceServer) CheckPids(ctx context.Context, req *pb.CheckPids) (*pb.PidVerifications, error) {

	log.Println("check pid called")

	ip := findIpFromPeerConnection(ctx)

	if ip == ""{
		return nil, fmt.Errorf("failed to get connection information")
	}

	var pidList []*pb.Pidstates


	for _,entry := range req.PidLaunchfileInfo{

		pidInt, err := strconv.Atoi(entry.Pids)
		
		if err != nil {
			fmt.Printf("Error converting string to int: %v\n", err)

			pidStateEntry := &pb.Pidstates{
				Pid: entry.Pids,
				State: "error",
			}
			pidList = append(pidList, pidStateEntry)
		}else{

			if utility.IsProcessActive(pidInt){
				pidStateEntry := &pb.Pidstates{
					Pid: entry.Pids,
					State: "running",
				}

				time := time.Now()

				s.ChildPids[entry.Launchfile] = childPidsValues{pid: int32(pidInt), time: time}
				pidList = append(pidList, pidStateEntry)
			}
		}
	}

	return &pb.PidVerifications{Pidstates: pidList}, nil
}


func (s *LogServiceServer) Shutdown(ctx context.Context, req *pb.Secret) (*pb.StatusResponse, error){

	log.Println("shutdown initiated")

	ip := findIpFromPeerConnection(ctx)

	if ip == ""{
		return nil, fmt.Errorf("failed to get connection information")
	}

	if status := !utility.IsValidJWT(req.GetMessage()); !status{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "JWT not valid token",
		}
		return statusResponse, fmt.Errorf("jwt is not a valid token")
	}

	keydata := utility.ReadFile(s.KeyPath)
	key, err := utility.PemToECDSAPub(string(keydata))

	if err != nil{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "There was an error",
		}
		return statusResponse, fmt.Errorf("there was an error")
	}


	status, err := utility.CheckTokenExpiry(req.GetMessage(), key)

	if status && err != nil{
		statusResponse := &pb.StatusResponse{
			Success: false,
			Message: "The token is wrong",
		}
		return statusResponse, fmt.Errorf("the token is wrong")
	}

	for _,entry := range s.ChildPids{
		utility.TerminateProcess(int(entry.pid))

		time.Sleep(500 * time.Millisecond)
	}

	go func() {
		
		time.Sleep(500 * time.Millisecond)
		
		log.Println("Shutting down server...")
		s.quit <- syscall.SIGTERM
	}()
	
	statusResponse := &pb.StatusResponse{
		Success: true,
		Message: "shutdown successfull",
	}

	return statusResponse, nil
}