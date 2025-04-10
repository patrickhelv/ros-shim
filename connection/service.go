package connection

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"ros2shim/utility"
	"syscall"
)

// function that starts the ros node given a launch
// Forks a seperate process which runs the ros node
// requires the workpackage_dir, the package/project name and the launch File
// returns (int) that represent the pid if successfull or returns -1 if not
func StartLaunch(rosNodeInfo RosNodeInfo, launchFile string, projectName string, directory string, venv bool) (int,error) {

	command := ""

	if venv{
		command = fmt.Sprintf("nohup bash -c 'source %s && source %s/install/setup.bash && ros2 launch %s %s' > /dev/null 2>&1 & echo $!", rosNodeInfo.venvPath, directory, projectName, launchFile)
	}else{
		command = fmt.Sprintf("nohup bash -c 'source %s/install/setup.bash && ros2 launch %s %s' > /dev/null 2>&1 & echo $!", directory, projectName, launchFile)
	}

	log.Println(command)

	cmd := exec.Command("bash", "-c", command)

	// Make the command a separate process
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		log.Printf("Error Unable to start the ros package %s with launch file %s, %s\n", directory, launchFile, err)
		return -1, fmt.Errorf("error Unable to start the ros package %s with launch file %s, %s", directory, launchFile, err)
	}

	return cmd.Process.Pid, nil
}

//  really important to you install ros2_humble in a specific folder like ros2 guide showed
// specify that you need to specify it from the home folder

// Builds the project
// requires the workpackage_dir, the package/project name
// requires the ros path which we assume is installed in your home directory
// returns (bool), where true represent a successfull build and false represents an error in the build
func BuildProject(rosNodeInfo RosNodeInfo, directory string, rosProjectName string, venv bool) bool {

	log.Println("build called")

	command := ""
	
	if venv{
		command = fmt.Sprintf(
			"cd %s && source %s && source %s && colcon build --packages-select %s",
			directory, rosNodeInfo.Ros2Path+"/install/setup.bash", rosNodeInfo.venvPath, rosProjectName)
	}else{

		command = fmt.Sprintf("cd %s && source %s && colcon build --packages-select %s",
			directory, rosNodeInfo.Ros2Path, rosProjectName)
	}

	log.Println(command)
	cmd := exec.Command("bash", "-c", command)

	err := cmd.Run()
	if err != nil {
		log.Printf("Error building the ros package %s, %s\n", rosNodeInfo.WorkpackageDirectory, err)
		return false
	}
	return true
}

// initializes the venv and install all the dependencies
func InitVenvAndRequirements(rosNodeInfo *RosNodeInfo, directory string) (bool){
	log.Println("venv called")

	rosNodeInfo.AddVenvPath(filepath.Join(directory, "venv", "bin", "activate"))
	requirementsTXT := filepath.Join(directory, "requirements.txt")


	// Command to create virtual environment if it doesn't exist

	if _, err := os.Stat(requirementsTXT); os.IsNotExist(err) {
		log.Println("No requirements.txt found, skipping installation of Python packages.")
		return true // Return true if no requirements.txt to handle, otherwise false if it's expected to be there
	}

	if _, err := os.Stat(rosNodeInfo.venvPath); os.IsNotExist(err) {
		log.Println("Virtual environment not found, creating one...")
		cmd := exec.Command("python3", "-m", "venv", directory + "/venv")
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to create virtual environment: %v\n", err)
			return false
		}
	}
	log.Printf("cd %s && source %s && pip install -r requirements.txt\n", directory, rosNodeInfo.venvPath)

	cmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && source %s && pip install -r requirements.txt --verbose", directory, rosNodeInfo.venvPath))
		
	err := cmd.Run()
	if err != nil {
		log.Printf("Error building the ros package %s, %s\n", rosNodeInfo.WorkpackageDirectory, err)
		return false
	}
	return true
	
}

// sources the ros project
func SourceProject(rosNodeInfo RosNodeInfo, directory string, venv bool) bool {
	command := ""

	if venv{
		command = fmt.Sprintf("cd %s && source %s && source %s", 
			directory, rosNodeInfo.venvPath, directory+"/install/setup.bash")
	}else{
		command = fmt.Sprintf("cd %s && source %s", 
			directory, directory+"/install/setup.bash")
	}

	cmd := exec.Command("bash", "-c", command)

	log.Println(command)

	err := cmd.Run()
	if err != nil {
		log.Printf("Error sourcing the ros package %s, %s\n", rosNodeInfo.WorkpackageDirectory, err)
		return false
	}
	return true
}

// Cleans the project
// requires the workpackage_dir
// returns (bool), where true represent a successfull clean and false represents an error in the cleaning process
func CleanProject(rosnodeinfo RosNodeInfo) bool {

	cmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && rm -rf build/ install/ log/", rosnodeinfo.WorkpackageDirectory))

	err := cmd.Run()
	if err != nil {
		log.Printf("Error cleaning the ros package %s, %s\n", rosnodeinfo.WorkpackageDirectory, err)
		return false
	}
	return true
}

// function that clones the ros project from git need to be hosted git server
// returns (true if sucessfully cloned)
// returns (false if not successfully cloned)
func CloneProject(giturl string, gitBranch string, directory string) bool{
	// Execute the git clone command
	cmd := exec.Command("git", "clone", giturl, "-b", gitBranch, directory)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("Error cloning repository: %s in %s\n", err, directory)
		return false
	}

	log.Println("Repository cloned successfully!")
	return true

}

// Shutdown the ros node with is associated pid
// returns (bool) true if successfull or false if not
func ShutDownChild(pid int) bool {
	return utility.TerminateProcess(pid)
}

// starts the python to DDS bridge
func StartPythonBridge(rosNodeInfo RosNodeInfo, topic string, endpoint string) (bool) {

	// source the ros2 installation folder

	command := fmt.Sprintf("source %s/install/setup.bash && python3 ./dds-bridge/ros2bridge.py --topic %s --endpoint %s", rosNodeInfo.Ros2Path, topic, endpoint)

	fmt.Println(command)

	cmd := exec.Command("bash", "-c", command)

	if err := cmd.Start(); err != nil {
		log.Printf("Unable to start python dds bridge using %s, topic %s and endpoint %s, with error %v", rosNodeInfo.Ros2Path, topic, endpoint, err)
		return false
	}

	log.Printf("Started process with PID %d\n", cmd.Process.Pid)

	return true
}

// updates the git project when the repository changes
func UpdateGitRepository(directory string) bool {

	log.Printf("Updating repository in %s\n", directory)


	cmd := exec.Command("git", "-C", directory, "pull", "--rebase")

	err := cmd.Run()

	if err != nil {
		log.Printf("Error updating repository: %s\n", err)
		return false
	}

	log.Println("Repository updated successfully!")
	return true
}
