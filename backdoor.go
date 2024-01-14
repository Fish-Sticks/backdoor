package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func popupMessage(message string) {
	pMyMessage, err := syscall.UTF16PtrFromString(message)
	pMyTitle, err2 := syscall.UTF16PtrFromString("Remote Message")

	if err != nil || err2 != nil {
		fmt.Println("Failed to appear message due to internal error")
		return
	}

	windows.MessageBox(windows.HWND(0), pMyMessage, pMyTitle, 0)
}

func callNTapi() {
	CurrentProcess := -1
	CurrentProcessHandle := *(*uintptr)(unsafe.Pointer(&CurrentProcess))

	BaseAddress := 0
	BaseAddressPtr := (uintptr)(unsafe.Pointer(&BaseAddress))

	RegionSize := 4096
	RegionSizePtr := (uintptr)(unsafe.Pointer(&RegionSize))

	AllocationType := uintptr(windows.MEM_COMMIT | windows.MEM_RESERVE)

	ntdll := windows.NewLazyDLL("ntdll.dll")
	ntdll.Load()

	NtAllocateVirtualMemory := ntdll.NewProc("NtAllocateVirtualMemory")

	fmt.Printf("NtAllocateVirtualMemory: %016X\n", NtAllocateVirtualMemory.Addr())

	status, _, _ := NtAllocateVirtualMemory.Call(CurrentProcessHandle, BaseAddressPtr, uintptr(0), RegionSizePtr, AllocationType, uintptr(0x40))

	if uint32(status) != uint32(windows.STATUS_SUCCESS) {
		fmt.Println("Failed to allocate memory VIA NtAllocateVirtualMemory!")
		return
	}

	fmt.Printf("RWX Memory allocated at %016X\n", *(*uintptr)(unsafe.Pointer(BaseAddressPtr)))
}

func GetBaseAddress() uintptr {
	zero := 0
	windowsNULL := *(**uint16)(unsafe.Pointer(&zero))

	var myHandle windows.Handle
	if err := windows.GetModuleHandleEx(0, windowsNULL, &myHandle); err != nil {
		fmt.Println("GetBaseAddress failed!")
		return 0
	}

	return uintptr(myHandle)
}

func RootHandler(response http.ResponseWriter, request *http.Request) {
	io.WriteString(response, `
	<main>
		<h1 style="color: red; text-align: center; font-size: 48px; font-weight: bold;">Welcome to my PC!</h1>
		<p>You can control my PC from this convienent web interface and execute rootkit from the web!</p>
		<p>This is completely <b>untraceable</b> so you can execute nasty viruses and not get caught!</p>
	</main>
	`)
}

func NotFoundHandler(response http.ResponseWriter, request *http.Request) {
	response.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(response, `
		<h1 style="color: red; text-align: center; font-size: 48px; font-weight: bold;">Page not found!</h1>
		<p>Couldn't find any data under the "%v" directory.</p>
	`, request.URL.Path)
}

func HelloHandler(response http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(response, "I am running at base address: 0x%016X", GetBaseAddress())
}

func MessageHandler(response http.ResponseWriter, request *http.Request) {
	io.WriteString(response, `
		<p>Welcome to the message handler! Here you will be able to issue remote messages to my PC and scare me!</p>

		<style>
			#message {
				width: 50vw;
				margin-left: 25vw;
			}
			#submit {
				margin-top: 1vw;
				margin-left: 45vw;
				padding: 10px 25px;
				font-weight: bold;
				background-color: lime;
				color: black;
			}
		</style>

		<iframe name="hiddenFrame" width="0" height="0" border="0" style="display: none;"></iframe>
		<form target="hiddenFrame" action="/sendmessage" method="post">
			<label for="message">Message:</label><br>
			<input id="message" name="message" type="text" value=""/><br>
			<input id="submit" type="submit" value="submit"/>
		</form>
	`)
}

func ReceiveMessageHandler(response http.ResponseWriter, request *http.Request) {
	if currentmethod := request.Method; currentmethod != "POST" {
		response.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(response, `
			<p>This route doesn't accept %v</p>
		`, currentmethod)

		return
	}

	if err := request.ParseForm(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		io.WriteString(response, `
			<p>An internal server error has occured.</p>
		`)
		return
	}

	result := request.Form.Get("message")
	if result == "" {
		response.WriteHeader(http.StatusBadRequest)
		io.WriteString(response, `
			<p>Expected message, got nil.</p>
		`)
		return
	}

	go popupMessage(result)
	response.Header().Set("Location", "/message")
}

func MiddleWare(response http.ResponseWriter, request *http.Request) {
	fmt.Printf("%v -> Received request from IP: %v | %v\n", request.URL.Path, request.RemoteAddr, request.Header.Get("X-Forwarded-For"))
	generateSite := func(handler http.HandlerFunc) {
		io.WriteString(response, `
		<!DOCTYPE html>
		<html>
			<head>
				<title>Birdy's PC</title>
			</head>

			<body>
				<style>
					body {
						background-color: black;
						color: lime;
						font-family: "Lucida Console", "Courier New", monospace;
					}
					
					footer {
						position: absolute;
						bottom: 0;
					}

					.mylinks {
						padding: 15px;
					}
				</style>

				<main>`)
		handler(response, request)
		io.WriteString(response, `
				</main>

				<footer>
					<div class="mylinks">
						<p style="margin-top: 50px;">Links:</p>
						<a href="/">Home.exe</a>
						<a href="/hello">Hello.exe</a>
						<a href="/message">Message.exe</a>
					</div>
				</footer>
			</body>
		</html>`)
	}

	switch currentDirectory := request.URL.Path; currentDirectory {
	case "/":
		generateSite(RootHandler)
	case "/hello":
		generateSite(HelloHandler)
	case "/message":
		generateSite(MessageHandler)
	case "/sendmessage":
		generateSite(ReceiveMessageHandler)
	default:
		fmt.Printf("Unknown directory indexed: %v\n", currentDirectory)
		generateSite(NotFoundHandler)
	}
}

func SetupWebServer() {
	http.HandleFunc("/", MiddleWare)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func main() {
	fmt.Println("Server is running!")

	username := os.Getenv("USERNAME")

	callNTapi()

	fmt.Println("Welcome to server " + username)
	SetupWebServer()
}
