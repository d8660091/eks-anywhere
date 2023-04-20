package e2e

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/eks-anywhere/internal/pkg/s3"
)

const (
	snowCredentialsS3Path  = "T_SNOW_CREDENTIALS_S3_PATH"
	snowCertificatesS3Path = "T_SNOW_CERTIFICATES_S3_PATH"
	snowDevices            = "T_SNOW_DEVICES"
	snowCPCidr             = "T_SNOW_CONTROL_PLANE_CIDR"
	snowCPCidrs            = "T_SNOW_CONTROL_PLANE_CIDRS"
	snowStaticIPCidrs      = "T_SNOW_STATIC_IP_CIDRS"
	snowStaticIPGateway    = "T_SNOW_STATIC_IP_GATEWAY"
	snowStaticIPSubnet     = "T_SNOW_STATIC_IP_SUBNET"
	snowCredsFile          = "EKSA_AWS_CREDENTIALS_FILE"
	snowCertsFile          = "EKSA_AWS_CA_BUNDLES_FILE"

	snowTestsRe       = `^.*Snow.*$`
	snowCredsFilename = "snow_creds"
	snowCertsFilename = "snow_certs"
)

var (
	snowCPCidrArray  []string
	snowCPCidrArrayM sync.Mutex

	snowStaticIPCidrArray  []string
	snowStaticIPCidrArrayM sync.Mutex
)

func init() {
	snowCPCidrArray = strings.Split(os.Getenv(snowCPCidrs), ",")
	snowStaticIPCidrArray = strings.Split(os.Getenv(snowStaticIPCidrs), ",")
}

// Note that this function cannot be called more than the the number of cidrs in the list.
func getSnowCPCidr() (string, error) {
	snowCPCidrArrayM.Lock()
	defer snowCPCidrArrayM.Unlock()

	if len(snowCPCidrArray) == 0 {
		return "", fmt.Errorf("no more snow control plane cidrs available")
	}
	var r string
	r, snowCPCidrArray = snowCPCidrArray[0], snowCPCidrArray[1:]
	return r, nil
}

func releaseSnowCPCidr(cpCidr string) {
	snowCPCidrArrayM.Lock()
	defer snowCPCidrArrayM.Unlock()

	snowCPCidrArray = append(snowCPCidrArray, cpCidr)
}

// Note that this function cannot be called more than the the number of cidrs in the list.
func getSnowStaticIPCidr() (string, error) {
	snowStaticIPCidrArrayM.Lock()
	defer snowStaticIPCidrArrayM.Unlock()

	if len(snowStaticIPCidrArray) == 0 {
		return "", fmt.Errorf("no more snow static IP cidrs available")
	}
	var r string
	r, snowStaticIPCidrArray = snowStaticIPCidrArray[0], snowStaticIPCidrArray[1:]
	return r, nil
}

func (e *E2ESession) setupSnowEnv(testRegex string) error {
	re := regexp.MustCompile(snowTestsRe)
	if !re.MatchString(testRegex) {
		return nil
	}

	e.testEnvVars[snowDevices] = os.Getenv(snowDevices)
	cpCidr, err := getSnowCPCidr()
	e.cleanups = append(e.cleanups, func() {
		e.logger.V(1).Info("Release control plane CIDR", "cidr", cpCidr, "instanceId", e.instanceId)
		releaseSnowCPCidr(cpCidr)
	})
	if err != nil {
		return err
	}
	e.testEnvVars[snowCPCidr] = cpCidr
	e.logger.V(1).Info("Assigned control plane CIDR to admin instance", "cidr", cpCidr, "instanceId", e.instanceId)

	if err := sendFileViaS3(e, os.Getenv(snowCredentialsS3Path), snowCredsFilename); err != nil {
		return err
	}
	if err := sendFileViaS3(e, os.Getenv(snowCertificatesS3Path), snowCertsFilename); err != nil {
		return err
	}
	e.testEnvVars[snowCredsFile] = "bin/" + snowCredsFilename
	e.testEnvVars[snowCertsFile] = "bin/" + snowCertsFilename

	staticIPRegex := regexp.MustCompile(".*StaticIP.*")
	if staticIPRegex.MatchString(testRegex) {
		if err := setStaticIPEnvVars(e.testEnvVars); err != nil {
			return err
		}
		e.logger.V(1).Info("StaticIP environment variables have been setup", "instanceId", e.instanceId)
	}

	return nil
}

func setStaticIPEnvVars(testEnvVars map[string]string) error {
	cidr, err := getSnowStaticIPCidr()
	if err != nil {
		return err
	}

	// get first ip from cidr
	startIP, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	lastIP := make(net.IP, len(startIP))
	copy(lastIP, startIP)
	for i := 0; i < len(ipnet.Mask); i++ {
		ipIdx := len(lastIP) - i - 1
		lastIP[ipIdx] = startIP[ipIdx] | ^ipnet.Mask[len(ipnet.Mask)-i-1]
	}
	testEnvVars["T_SNOW_IPPOOL_IPSTART"] = startIP.String()
	testEnvVars["T_SNOW_IPPOOL_IPEND"] = lastIP.String()
	testEnvVars["T_SNOW_IPPOOL_GATEWAY"] = os.Getenv(snowStaticIPGateway)
	testEnvVars["T_SNOW_IPPOOL_SUBNET"] = os.Getenv(snowStaticIPSubnet)

	return nil
}

func sendFileViaS3(e *E2ESession, s3Path string, filename string) error {
	if err := s3.DownloadToDisk(e.session, s3Path, e.storageBucket, "bin/"+filename); err != nil {
		return err
	}

	err := e.uploadRequiredFile(filename)
	if err != nil {
		return fmt.Errorf("failed to upload file (%s) : %v", filename, err)
	}

	err = e.downloadRequiredFileInInstance(filename)
	if err != nil {
		return fmt.Errorf("failed to download file (%s) in admin instance : %v", filename, err)
	}
	return nil
}
