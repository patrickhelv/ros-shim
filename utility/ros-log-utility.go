package utility

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"log"
	"io"
)

// Finds the log file in logfilepath using the ros node pid
// returns the full fileName if successfull
// returns "-1" if fileName is not found
func GetLogFilename(pid string, logfilepath string) (string, error) {
	files, err := os.ReadDir(logfilepath)
	if err != nil {
		log.Printf("Error reading log directory")
		return "", fmt.Errorf("error reading log directory, %v", err)
	}

	for _, file := range files { // searches all the files for the pid
		if strings.Contains(file.Name(), pid) && filepath.Ext(file.Name()) == ".log" {
			return file.Name(), nil
		}
	}

	return "", fmt.Errorf("error did not find the filename")
}

// find the logfile in a directory
// returns "" if there are error
// returns the path if successfull
func FindLogFile(rootDir, pid string, timestamp string) (string, error) {
	var logFilePath string
	found := false

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) (error) {
		if err != nil {
			return err
		}
		if d.IsDir(){
			if strings.Contains(d.Name(), timestamp) && strings.HasSuffix(d.Name(), pid) && !found {
				temp, err := findLogInDir(path)
				if err == nil {
					logFilePath = temp
					found = true
					return filepath.SkipDir
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("Error : %v", err)
		return "", fmt.Errorf("error : %v", err)
	}

	if logFilePath == "" {
		log.Println("log file not found")
		return "", fmt.Errorf("log file not found")
	}

	return logFilePath, nil
}

// finds the log file in a directory
// return the filepath in directory, nil if successfull
// returns "", error if unsuccessfull
func findLogInDir(dirPath string) (string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".log") {
			return filepath.Join(dirPath, file.Name()), nil
		}
	}
	return "", fmt.Errorf("no .log file found in directory: %s", dirPath)
}

// extracts the pid from a log file path
// returns "" if not successfull
// returns the match if successfull
func ExtractPIDFromLog(logFilePath string) (string, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Printf("error opening the log file, %v", err)
		return "", fmt.Errorf("error opening the log file, %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	regex := regexp.MustCompile(`process started with pid \[(\d+)\]`)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := regex.FindStringSubmatch(line); len(matches) > 1 {
			return matches[1], nil // Return the extracted PID
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error : %v", err)
		return "", fmt.Errorf("error : %v", err)
	}
	log.Printf("Error : %v", err)

	return "", fmt.Errorf("error : %v", err)
}

// function that enables us to read from the log file
// in default ros the path for the log files is ~/.ros/logs
// it will read the last 10 lines from the log files and return them
// returns []string (the last 10 lines)
// returns nil if there is an error
func ReadFromLog(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening file:", err)
		return nil
	}
	defer file.Close()

	// Seek to the end of the file
	stat, err := file.Stat()
	if err != nil {
		log.Println("Error getting file stat:", err)
		return nil
	}

	fileSize := stat.Size()
	bufferSize := 1024
	var pos int64 = fileSize - int64(bufferSize)
	if pos < 0 {
		pos = 0
	}

	// Adjust pos based on actual log file characteristics

	lines := []string{}
	for {
		if pos == 0 {
			file.Seek(0, io.SeekStart)
		} else {
			file.Seek(pos, io.SeekStart)
		}

		buffer := make([]byte, bufferSize)
		bytesRead, err := file.Read(buffer)
		if err != nil {
			log.Println("Error reading file:", err)
			return nil
		}

		buffer = buffer[:bytesRead]
		tmpLines := []string{}
		start := 0
		for i, b := range buffer {
			if b == '\n' {
				line := string(buffer[start:i])
				tmpLines = append(tmpLines, line)
				start = i + 1
			}
		}

		// Prepend to handle reverse reading
		lines = append(tmpLines, lines...)

		// Break if we have read enough lines or reached the start of the file
		if len(lines) >= 10 || pos == 0 {
			break
		}

		// Move back for the next read
		newPos := pos - int64(bufferSize)
		if newPos < 0 {
			bufferSize += int(newPos) // Adjust buffer size for the last read
			pos = 0                   // Set to start for the next iteration
		} else {
			pos = newPos
		}
	}

	return lines
}
