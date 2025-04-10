package connection

import (
	"fmt"
	"ros2shim/utility"
)

type RosNodeInfo struct {
	WorkpackageDirectory string   
	LaunchFiles          []string 
	Ros2Path             string   
	Port                 string
	LogPath 			 string
	venvPath 			 string
	power 				 bool
}

func (rosNodeInfo *RosNodeInfo) Init(workpackageDirectory string, launchFiles []string, ros2Path string ,port string, logpath string, power bool) {

	rosNodeInfo.WorkpackageDirectory = workpackageDirectory
	rosNodeInfo.LaunchFiles = launchFiles
	rosNodeInfo.Ros2Path = ros2Path
	rosNodeInfo.Port = port
	rosNodeInfo.LogPath = logpath
	rosNodeInfo.power = power
}

func (RosNodeInfo *RosNodeInfo) AddVenvPath(venvpath string){
	RosNodeInfo.venvPath = venvpath
}

// read the config file 
// the root path is the name of the installation folder you define from ~/home
// the log path is the name of log folder you define over the installation folder
func (rosNodeInfo *RosNodeInfo) ReadConfigFile(controlConfigFile string) ([]string, error) {

	fmt.Println("reading config file..")

	const CONFIG_ROS2_PATH = "ROS2_PATH" // Full path
	const CONFIG_ROOT_PATH = "ROOT_PATH"
	const CONFIG_LOG_PATH = "LOG_PATH"
	const CONFIG_ROS_LOG_PATH = "ROS_LOG_PATH"
	const CONFIG_CA_PEM = "CA_PEM"
	const CONFIG_CA_CLIENT_CERT = "CA_CLIENT_CERT"
	const CONFIG_CA_CLIENT_KEY = "CA_CLIENT_KEY"
	const CONFIG_CA_SERVER_CERT = "CA_SERVER_CERT"
	const CONFIG_CA_SERVER_KEY = "CA_SERVER_KEY"
	const CONFIG_SERVER_NAME = "NODE_PORT_DNS"
	const CONFIG_SECRET = "SECRET"
	const CONFIG_SECRET_KEY = "SECRET_KEY"
	const CONFIG_POWER = "POWER"
	const CONFIG_DDS_BRIDGE = "DDS_BRIDGE"
	const CONFIG_TOPIC = "TOPIC"
	const CONFIG_ENDPOINT = "ENDPOINT"

	var result []string

	// retrieves the value from the configuration file
	fmt.Println(controlConfigFile)
	configFileMap, err := utility.ReadConfigFile(controlConfigFile)
	if configFileMap == nil {
		fmt.Printf("Error reading the configuration file, %s", err)
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_ROS2_PATH]; exists {
		result = append(result, val)
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_ROOT_PATH]; exists { 
		result = append(result, val) // project path path 
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_LOG_PATH]; exists {
		result = append(result, val) // log path 
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_ROS_LOG_PATH]; exists {
		result = append(result, val) // ros log path 
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_CA_PEM]; exists {
		result = append(result, val) // ca perm
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_CA_CLIENT_CERT]; exists {
		result = append(result, val) // ca client cert
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_CA_CLIENT_KEY]; exists {
		result = append(result, val) // ca client key
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_CA_SERVER_CERT]; exists {
		result = append(result, val) // ca server key
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_CA_SERVER_KEY]; exists {
		result = append(result, val) // ca server key
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_SERVER_NAME]; exists {
		result = append(result, val)
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_SECRET]; exists {
		result = append(result, val) // token
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_SECRET_KEY]; exists {
		result = append(result, val) // token key
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_POWER]; exists {
		result = append(result, val) // token key
	} else {
		return nil, err
	}

	if val, exists := configFileMap[CONFIG_DDS_BRIDGE]; exists { // if dds bridge then need a ros2 topic and web server if not then not needed
		result = append(result, val) // dds bridge

		if val, exists := configFileMap[CONFIG_TOPIC]; exists {
			result = append(result, val) // ros2 topic to subscribe to
		} else {
			return nil, fmt.Errorf("DDS bridge configuration missing TOPIC, please add a TOPIC to subscribe to")
		}

		if val, exists := configFileMap[CONFIG_ENDPOINT]; exists {
			result = append(result, val) // ip web server 
		} else {
			return nil, fmt.Errorf("DDS bridge configuration missing IP_WEB_SERVER, please add the ip of web server to send the information to")
		}

	}else{
		result = append(result, "false") // dds bridge
	}


	installationPath := configFileMap[CONFIG_ROS2_PATH] // installation path 

	fmt.Printf("Installation path : %s \n", installationPath)

	return result, nil 
}