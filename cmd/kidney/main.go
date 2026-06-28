package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vsyaco/kidney/internal/doctor"
	"github.com/vsyaco/kidney/internal/domain"
	"github.com/vsyaco/kidney/internal/library"
	"github.com/vsyaco/kidney/internal/server"
	"github.com/vsyaco/kidney/internal/transport"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	transports := defaultTransports()
	service := library.NewService(transports)

	switch os.Args[1] {
	case "serve":
		runServe(service, transports, os.Args[2:])
	case "devices":
		runDevices(service, os.Args[2:])
	case "doctor":
		runDoctor(transports, os.Args[2:])
	case "list":
		runList(service, os.Args[2:])
	case "upload":
		runUpload(service, os.Args[2:])
	case "download":
		runDownload(service, os.Args[2:])
	case "delete":
		runDelete(service, os.Args[2:])
	case "rename":
		runRename(service, os.Args[2:])
	case "unmount":
		runUnmount(service, os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func runServe(service *library.Service, transports []domain.Transport, args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	port := flags.Int("port", 8765, "localhost port")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	app, err := server.New(service)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	httpServer := &http.Server{
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownContext.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		unmountExisting(ctx, transports)

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}()

	fmt.Printf("Kidney is running at http://%s\n", listener.Addr().String())
	if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func runDevices(service *library.Service, args []string) {
	flags := flag.NewFlagSet("devices", flag.ExitOnError)
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	devices, err := service.Devices(ctx)
	if err != nil && len(devices) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n", library.FriendlyError(err))
		os.Exit(1)
	}

	if *jsonOutput {
		writeJSON(devices)
		return
	}

	if len(devices) == 0 {
		fmt.Println("No Kindle devices detected.")
		return
	}

	for _, device := range devices {
		fmt.Printf(
			"%s\tbackend=%s\tmodel=%s\tserial=%s\tdocuments=%s\t%s\n",
			device.Name,
			device.Backend,
			device.Model,
			device.Serial,
			device.DocumentsPath,
			device.Message,
		)
	}
}

func runDoctor(transports []domain.Transport, args []string) {
	flags := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := doctor.Context(context.Background())
	defer cancel()
	defer func() {
		unmountContext, unmountCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer unmountCancel()
		unmountExisting(unmountContext, transports)
	}()

	report := doctor.Run(ctx, transports)
	if *jsonOutput {
		writeJSON(report)
		return
	}

	fmt.Print(doctor.Print(report))
	fmt.Printf("\nSupported files: %s\n", doctor.SupportedExtensionsLine())
}

func runList(service *library.Service, args []string) {
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	jsonOutput := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	files, err := service.List(ctx)
	if err != nil {
		exitWithFriendlyError(err)
	}

	if *jsonOutput {
		writeJSON(files)
		return
	}

	if len(files) == 0 {
		fmt.Println("No supported files found.")
		return
	}

	for _, file := range files {
		fmt.Printf("%s\t%d\t%s\n", firstNonEmpty(file.Path, file.Name), file.SizeBytes, file.Modified.Format(time.RFC3339))
	}
}

func runUpload(service *library.Service, args []string) {
	flags := flag.NewFlagSet("upload", flag.ExitOnError)
	deviceName := flags.String("name", "", "file name on Kindle")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: kidney upload [-name device-name] <local-file>")
		os.Exit(2)
	}

	localPath := flags.Arg(0)
	file, err := os.Open(localPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	name := *deviceName
	if name == "" {
		name = fileName(localPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	book, err := service.Upload(ctx, file, name)
	if err != nil {
		exitWithFriendlyError(err)
	}

	fmt.Printf("Uploaded %s (%d bytes)\n", book.Name, book.SizeBytes)
}

func runDownload(service *library.Service, args []string) {
	flags := flag.NewFlagSet("download", flag.ExitOnError)
	outputPath := flags.String("output", "", "local output path")
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: kidney download [-output local-file] <file-name>")
		os.Exit(2)
	}

	fileID := flags.Arg(0)
	localPath := *outputPath
	if localPath == "" {
		localPath = fileName(fileID)
	}

	file, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	book, err := service.Download(ctx, fileID, file)
	if err != nil {
		_ = os.Remove(localPath)
		exitWithFriendlyError(err)
	}

	fmt.Printf("Downloaded %s to %s (%d bytes)\n", firstNonEmpty(book.Path, book.Name), localPath, book.SizeBytes)
}

func runDelete(service *library.Service, args []string) {
	flags := flag.NewFlagSet("delete", flag.ExitOnError)
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: kidney delete <file-name>")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := service.Delete(ctx, flags.Arg(0)); err != nil {
		exitWithFriendlyError(err)
	}

	fmt.Println("Deleted.")
}

func runRename(service *library.Service, args []string) {
	flags := flag.NewFlagSet("rename", flag.ExitOnError)
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	if flags.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: kidney rename <old-name> <new-name>")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	book, err := service.Rename(ctx, flags.Arg(0), flags.Arg(1))
	if err != nil {
		exitWithFriendlyError(err)
	}

	fmt.Printf("Renamed to %s\n", book.Name)
}

func runUnmount(service *library.Service, args []string) {
	flags := flag.NewFlagSet("unmount", flag.ExitOnError)
	if err := flags.Parse(args); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := service.Unmount(ctx); err != nil {
		exitWithFriendlyError(err)
	}

	fmt.Println("Unmounted.")
}

func defaultTransports() []domain.Transport {
	return []domain.Transport{
		transport.NewDiskTransport(),
		transport.NewMTPTransport(),
	}
}

type existingMountTransport interface {
	UnmountAll(ctx context.Context) error
}

func unmountExisting(ctx context.Context, transports []domain.Transport) {
	for _, item := range transports {
		mountTransport, ok := item.(existingMountTransport)
		if !ok {
			continue
		}

		if err := mountTransport.UnmountAll(ctx); err != nil {
			log.Printf("unmount failed for %s: %v", item.Name(), err)
		}
	}
}

func exitWithFriendlyError(err error) {
	fmt.Fprintln(os.Stderr, library.FriendlyError(err))
	os.Exit(1)
}

func fileName(path string) string {
	for index := len(path) - 1; index >= 0; index-- {
		if path[index] == '/' || path[index] == '\\' {
			return path[index+1:]
		}
	}

	return path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

func writeJSON(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println(`Usage:
  kidney serve [-port 8765]
  kidney devices [-json]
  kidney doctor [-json]
  kidney list [-json]
  kidney upload [-name device-name] <local-file>
  kidney download [-output local-file] <file-name>
  kidney delete <file-name>
  kidney rename <old-name> <new-name>
  kidney unmount

Commands:
  serve    Start the local web UI.
  devices  Print detected Kindle devices.
  doctor   Check USB visibility and MTP/disk tooling.
  list     List supported files on Kindle.
  upload   Upload one local file to Kindle.
  download Download one Kindle file.
  delete   Delete one file from Kindle.
  rename   Rename one file on Kindle.
  unmount  Unmount app-owned MTP mount.`)
}
