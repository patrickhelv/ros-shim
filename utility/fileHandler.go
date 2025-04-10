package utility

import(
	"bufio"
	"os"
	"strings"
	"log"
	"fmt"
	"time"
	"path/filepath"
)

// Reads a specified config files
// returns (nil, error) if the an error happens while reading
// returns (map[string]string, nil) if the reading of the file is successfull
func ReadConfigFile(filePath string) (map[string]string, error) {

	config := make(map[string]string)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Iterate through each line in the file
	for scanner.Scan() {
		line := scanner.Text()

		// reading key-value pairs separated by '='
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// method to check if a value exists in a list,
// returns true if it exists,
// returns false if it does not.
func ExistsIn(value string, list []string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// Checks if the path exists or not
// returns (false, error) if the path does not exist
// returns (false, nil) if there is any other error
// returns (true, nil) if the paths exists
func Exists(path string) (bool, error) {

	_, err := os.Stat(path)
	if err == nil {
		return true, nil // path exists
	}

	if os.IsNotExist(err) {
		return false, err
	}

	return false, nil
}

// checks if the folder is empty or not
// returns (false, error=nil/error) if the folder is empty
// returns (true, error=nil) if there are folders
func IsFolderNotEmpty(folderPath string) (bool, error) {
	dir, err := os.Open(folderPath)
	if err != nil {
		fmt.Printf("Error when opening folder %s, %s\n", folderPath, err)
		return false, err
	}

	defer dir.Close()

	_, err = dir.Readdir(1) //reads the dir and if returns nil means that there are files in the folder
	if err != nil {
		fmt.Printf("Error the folder %s is empty, %s\n", folderPath, err)
		return false, err
	}

	return true, nil
}

// Checks the directory if it containes a launch directory,
// it is mandatory for a ros project to include a launch directory,
// that contains the launch files.
// returns (bool, error)
// returns false, err if we do not find the launch directory
// returns true, nil if we find the launch directory
func CheckLaunchDir(workpackageDirectory string, launchFile string) (bool, error) {
	found := false

	err := filepath.Walk(workpackageDirectory, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		// Check if the current path is a directory and its name is "launch"
		if info.IsDir() && info.Name() == "launch" {
			launchFilePath := filepath.Join(path, launchFile)
			if _, err := os.Stat(launchFilePath); err == nil {
				found = true
				return filepath.SkipDir // Stop walking further into this directory
			}else if os.IsNotExist(err){
				return nil
			}else{
				return err
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error walking through directory:", err)
		return false, err
	}
	return found, err

}


// creates the new log file from the log directory and the logfilename
// logFatalf is unsuccessfull
func InitLogFile(logDir string, logFileName string) {

	// This format (2006-01-02_15-04-05) is Go's reference time format
	currentTime := time.Now().Format("2006-01-02_15-04-05")
	fullname := logFileName + "_" + currentTime + ".log"

	// Ensure the directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// Combine dir and filename
	fullLogPath := logDir + "/" + fullname

	// Open or create the log file
	logFile, err := os.OpenFile(fullLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Set the output of the log package to the new log file.
	log.SetOutput(logFile)

	// Example log entry
	log.Println("This is a test log entry")
}

// verifies the ros_projects existance before importing them from git
// checks also if there are launch files
// returns (false, error) if the ros work folder does not exists
// returns (false, error) if the ros work folder is not populated with files and folders
// returns (false, error) if the ros work folder does not contain a launch folder
// returns (true, nil) if the both the ros work folder exists and it is populated with files
func VerifyProjectExistence(workpackageDirectory string, launchFile string) (bool, error) {

	RosWorkFolderExists, err := Exists(workpackageDirectory)
	if err != nil {
		return false, err
	}

	ProjectNotEmpty, err := IsFolderNotEmpty(workpackageDirectory)
	if err != nil {
		return false, err
	}

	// find the launchFolder
	LaunchFileExist, err := CheckLaunchDir(workpackageDirectory, launchFile)
	if err != nil {
		return false, err
	}

	return RosWorkFolderExists && ProjectNotEmpty && LaunchFileExist, nil
}

// checks if the ros project is already build
func IsProjectBuilt(workpackageDir string) bool{

	buildPath := workpackageDir + "/build"

	buildExist, _ := Exists(workpackageDir + "/build")
	notEmpty,_ := IsFolderNotEmpty(buildPath)

	if buildExist && notEmpty{
		installExist,_ := Exists(workpackageDir + "/install/setup.bash")
		notEmpty,_ := IsFolderNotEmpty(buildPath)
		
		if installExist && notEmpty{
			return true
		}
	}

	return false

}

func RemoveFile(path string) bool {

	err := os.Remove(path + ".yaml")

	if err != nil {
		fmt.Printf("Did not delete file %s", path)
		return false
	}

	return true
}

func ReadFile(path string) ([]byte){

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read certificate file: %s", err)
	}

	return data
}

// SplitHostPort splits a string of the format "host:port" into separate host and port strings.
func SplitHostPort(hostPort string) (string, string, error) {
    parts := strings.Split(hostPort, ":")
    if len(parts) != 2 {
        return "", "", fmt.Errorf("invalid input: expected format 'host:port'")
    }
    return parts[0], parts[1], nil
}

// Function to check if 'requirements.txt' exists in the specified directory.
// Returns true if the file exists, false otherwise.
func CheckRequirementsExists(directory string) bool {
    // Construct the path to 'requirements.txt'
    reqPath := filepath.Join(directory, "requirements.txt")

    // Use os.Stat to get the file info
    _, err := os.Stat(reqPath)
    if err != nil {
        if os.IsNotExist(err) {
            // File does not exist
            log.Printf("No requirements.txt found in: %s\n", directory)
            return false
        }
        // Other error
        log.Printf("Error checking for requirements.txt in: %s, error: %v\n", directory, err)
        return false
    }
    // File exists
    log.Printf("Found requirements.txt in: %s\n", directory)
    return true
}