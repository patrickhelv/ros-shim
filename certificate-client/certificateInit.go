package certificateclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"time"
	"os"
	"os/exec"
	"fmt"
	"net"

	"ros2shim/utility"

	pb "ros2shim/protoexport/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	maxRetries        = 100
	connectionTimeout = 10 * time.Minute
)

// GetHostname retrieves the hostname of the system.
func GetHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %v", err)
	}
	return hostname, nil
}

// CheckAvahiDaemon checks if avahi-daemon is running.
func CheckAvahiDaemon() (bool) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", "avahi-daemon")
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// avahi-daemon is not active
			return false
		}
		log.Printf("failed to check avahi-daemon status: %v\n", err)
		return false
	}
	return true
}


// IsHostLocalActive checks if host.local is active in DNS.
func IsHostLocalActive(hostname string) (bool) {
	_, err := net.LookupHost(hostname + ".local")
	if err != nil {
		if _, ok := err.(*net.DNSError); ok {
			// DNS error occurred
			return false
		}
		log.Printf("failed to lookup host.local: %v\n", err)
		return false
	}
	return true
}

// Configures the certificates for gRPC connections
func SetUpCertificate(caPemPath string, clientCertPath string, clientKeyPath string, serverName string, messagePath string) string {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a certificate pool from the CA
	certPool := x509.NewCertPool()
	caPem := utility.ReadFile(caPemPath)
	if ok := certPool.AppendCertsFromPEM(caPem); !ok {
		log.Fatalf("failed to append ca certs\n")
	}

	// Load client's certificate and private key
	clientPem := utility.ReadFile(clientCertPath)
	clientKey := utility.ReadFile(clientKeyPath)

	clientCert, err := tls.X509KeyPair(clientPem, clientKey)
	if err != nil {
		log.Fatalf("could not load client key pair: %s\n", err)
	}

	hostname,_,err := utility.SplitHostPort(serverName)

	if err != nil{
		log.Fatalf("The servername is in the wrong format DNS:PORT")
	}

	// Create a new TLS credentials based on the certificate pool and the client certs
	creds := credentials.NewTLS(&tls.Config{
		ServerName:   hostname, // Name of the server
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
	})

	var conn *grpc.ClientConn
	retryCount := 0
	startTime := time.Now()
	maxRetryDuration := 8 * time.Minute

	log.Println("Starting connecting to grpc server...")

	for time.Since(startTime) < maxRetryDuration {
		// Dial the server with the TLS credentials
		conn, err = grpc.DialContext(ctx, serverName, grpc.WithTransportCredentials(creds))
		log.Printf("Dial context on server %s\n", serverName)
		
		if err != nil {	
			log.Printf("did not connect: %v\n", err)
			retryCount++
			if retryCount < 10 {
				log.Printf("Retrying in 1 minute .. %v\n", time.Minute)
				time.Sleep(1 * time.Minute)
				continue
			} else {
				log.Println("Max retries reached")
				time.Sleep(connectionTimeout)
				retryCount = 0
			}

		}else{
			log.Println("Successfully connected to the gRPC server")
			break
		}
		
	}

	if conn == nil {
		log.Println("Failed to connect within the maximum retry duration")
		return ""
	}

	defer conn.Close()

	msgbyte := utility.ReadFile(messagePath)
	message := string(msgbyte)

	host, err := GetHostname()
	
	if err != nil{
		log.Fatalf("Error fetching hostname\n")
	}

	if !CheckAvahiDaemon() && !IsHostLocalActive(host){
		log.Printf("Error either the avahi daemon is not running or the local host is not active\n")
	}

	// Prepare your request with the secret
	req := &pb.Secret{
		Message: message,
		Dns: host+".local",
	}

	log.Printf("Sending request over to server %s, request : ip : %s, message: %s\n", serverName, req.Dns, req.Message)

	client := pb.NewRegisterRosShimClient(conn)

	resp, err := client.Register(context.Background(), req)

	if err != nil {
		log.Fatalf("error when calling register function: %v\n", err)
	}

	if resp.Success {
		log.Printf("The port is %s\n", resp.Port)
		return resp.Port
	}

	log.Println("the response from the server was not successfull")

	return ""
}
