package main

import (
	"log"
	"fmt"
	"strconv"

	"ros2shim/connection"
	"ros2shim/utility"
	"ros2shim/certificate-client"
)

func main() {
	// specify this from where you run your program no from main
	configFiles := "./config-files/nodeShimConfig.txt"

	fmt.Println("start..")

	rosNodeInfo := &connection.RosNodeInfo{}
	var launchFiles []string

	config, _ := rosNodeInfo.ReadConfigFile(configFiles)

	ros2path := config[0]
	workpackageDirectory := config[1]
	logPath := config[2]
	roslogPath := config[3]
	caPemPath := config[4]
	clientCertPath := config[5]
	clientKeyPath := config[6]
	serverCertPath := config[7]
	serverCertKey := config[8]
	serverName := config[9]
	messagePath := config[10]
	messagePathKey := config[11]
	powerstr := config[12]
	dds_bridge := config[13]

	utility.InitLogFile(logPath, "nodecontroller")

	if ros2path == "" {
		log.Println("The ros2 path is not found")
		return
	}

	boolVal, err := strconv.ParseBool(dds_bridge)
	if err != nil {
		log.Printf("error parsing boolean value for key %s: %v", dds_bridge, err)
	}

	log.Println("Setting up certificates")
	port := certificateclient.SetUpCertificate(caPemPath, clientCertPath, clientKeyPath, serverName, messagePath)

	if port == ""{
		log.Println("Error, the port was not specified by the kubernetes controller")
		return
	}
	power := false

	if power, err = strconv.ParseBool(powerstr); err != nil{
		log.Println("Error, could not parse the string to a boolean")
	}

	rosNodeInfo.Init(workpackageDirectory, launchFiles, ros2path, port, roslogPath, power)

	logservice := &connection.LogServiceServer{}
	logservice.UpdateRosNodeInfo(rosNodeInfo)

	log.Printf("ros2 path : %s", logservice.RosNodeinfo.Ros2Path)

	if(boolVal){
		topic := config[14]
		endpoint := config[15]
			
		if !connection.StartPythonBridge(*rosNodeInfo, topic, endpoint){
			log.Printf("Unable to start dds bridge")
		}
	}
		
	logservice.StartGrpcCommunication(port, serverCertPath, serverCertKey, messagePathKey, caPemPath)

}
