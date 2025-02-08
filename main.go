package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	resourceName   = "pcie.com/device"
	serverSock     = v1beta1.DevicePluginPath + "pcie.sock"
	pciDevicesPath = "/sys/bus/pci/devices"
)

type PCIDevicePlugin struct {
	devices []*v1beta1.Device
	server  *grpc.Server
}

func NewPCIDevicePlugin() *PCIDevicePlugin {
	devices := discoverPCIDevices()
	return &PCIDevicePlugin{
		devices: devices,
	}
}

func discoverPCIDevices() []*v1beta1.Device {
	var devices []*v1beta1.Device

	files, err := os.ReadDir(pciDevicesPath)
	if err != nil {
		log.Fatalf("Error reading PCI devices directory: %v", err)
	}

	for _, file := range files {
		vendorPath := filepath.Join(pciDevicesPath, file.Name(), "vendor")
		devicePath := filepath.Join(pciDevicesPath, file.Name(), "device")

		vendor, err := os.ReadFile(vendorPath)
		if err != nil {
			log.Printf("Error reading vendor file for device %s: %v", file.Name(), err)
			continue
		}

		deviceID, err := os.ReadFile(devicePath)
		if err != nil {
			log.Printf("Error reading device file for device %s: %v", file.Name(), err)
			continue
		}

		id := strings.TrimSpace(file.Name())
		devices = append(devices, &v1beta1.Device{
			ID:     id,
			Health: v1beta1.Healthy,
		})
		log.Printf("Discovered device: %s, Vendor: %s, Device ID: %s", id, vendor, deviceID)
	}

	return devices
}

func (dp *PCIDevicePlugin) Start() error {
	log.Println("Removing stale socket file if exists...")
	// Remove any stale socket file
	if err := os.Remove(serverSock); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("could not remove stale socket file %s: %v", serverSock, err)
	}

	listener, err := net.Listen("unix", serverSock)
	if err != nil {
		return fmt.Errorf("could not listen on %s: %v", serverSock, err)
	}
	log.Println("Socket created, starting gRPC server...")

	dp.server = grpc.NewServer()
	v1beta1.RegisterDevicePluginServer(dp.server, dp)

	go func() {
		if err := dp.server.Serve(listener); err != nil {
			log.Fatalf("could not start gRPC server: %v", err)
		}
	}()

	// Give the server a bit more time to start
	time.Sleep(10 * time.Second)

	log.Println("Connecting to gRPC server...")
	conn, err := grpc.Dial(serverSock, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("could not connect to gRPC server: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to gRPC server")

	return nil
}

func (dp *PCIDevicePlugin) Stop() error {
	if dp.server != nil {
		dp.server.Stop()
		dp.server = nil
	}
	return nil
}

func (dp *PCIDevicePlugin) Register(kubeletEndpoint string) error {
	log.Println("Connecting to kubelet...")
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("could not connect to kubelet: %v", err)
	}
	defer conn.Close()

	client := v1beta1.NewRegistrationClient(conn)
	req := &v1beta1.RegisterRequest{
		Version:      v1beta1.Version,
		Endpoint:     filepath.Base(serverSock),
		ResourceName: resourceName,
	}

	log.Println("Registering with kubelet...")
	if _, err := client.Register(context.Background(), req); err != nil {
		return fmt.Errorf("could not register device plugin: %v", err)
	}
	log.Println("Successfully registered with kubelet")

	return nil
}

func (dp *PCIDevicePlugin) ListAndWatch(e *v1beta1.Empty, s v1beta1.DevicePlugin_ListAndWatchServer) error {
	if err := s.Send(&v1beta1.ListAndWatchResponse{Devices: dp.devices}); err != nil {
		log.Printf("Error sending ListAndWatch response: %v", err)
		return err
	}

	for {
		time.Sleep(10 * time.Second)
	}
}

func (dp *PCIDevicePlugin) Allocate(ctx context.Context, reqs *v1beta1.AllocateRequest) (*v1beta1.AllocateResponse, error) {
	responses := &v1beta1.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		log.Printf("Processing request %s", req.String())
		response := &v1beta1.ContainerAllocateResponse{
			Devices: []*v1beta1.DeviceSpec{
				{
					ContainerPath: "/dev/null",
					HostPath:      "/dev/null",
					Permissions:   "mrw",
				},
			},
		}
		responses.ContainerResponses = append(responses.ContainerResponses, response)
	}
	return responses, nil
}

func (dp *PCIDevicePlugin) GetDevicePluginOptions(ctx context.Context, e *v1beta1.Empty) (*v1beta1.DevicePluginOptions, error) {
	return &v1beta1.DevicePluginOptions{}, nil
}

func (dp *PCIDevicePlugin) PreStartContainer(ctx context.Context, req *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	return &v1beta1.PreStartContainerResponse{}, nil
}

func (dp *PCIDevicePlugin) GetPreferredAllocation(ctx context.Context, req *v1beta1.PreferredAllocationRequest) (*v1beta1.PreferredAllocationResponse, error) {
	// Implement your preferred allocation logic here
	return &v1beta1.PreferredAllocationResponse{}, nil
}

func main() {
	kubeletEndpoint := "unix:///var/lib/kubelet/device-plugins/kubelet.sock"
	dp := NewPCIDevicePlugin()

	if err := dp.Start(); err != nil {
		log.Fatalf("Could not start device plugin: %v", err)
	}
	defer func() {
		if err := dp.Stop(); err != nil {
			log.Printf("Error stopping device plugin: %v", err)
		}
	}()
	log.Println("Started device plugin")

	if err := dp.Register(kubeletEndpoint); err != nil {
		log.Fatalf("Could not register device plugin: %v", err)
	}

	select {}
}
