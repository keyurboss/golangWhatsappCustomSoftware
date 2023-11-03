package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/golangWhatsappCustomSoftware/validator"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type Config struct {
	UseTextMessage                                   bool   `json:"useTextMessage" validate:"boolean"`
	AppendMessageToImage                             bool   `json:"appendMessageToImage" validate:"boolean"`
	ReadMessageFromCsv                               bool   `json:"readMessageFromCsv" validate:"boolean"`
	Message                                          string `json:"message"`
	AddMinimumDelayInSecondsAfterSuccessfullMesssage int    `json:"addMinimumDelayInSecondsAfterSuccessfullMesssage" validate:"required"`
	BasePathForAssets                                string `json:"basePathForAssets"`
}

var currentDir = ""
var ThisConfig = new(Config)
var client *whatsmeow.Client

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	case *events.Connected:
		println("Client Connected")
	default:
		fmt.Printf("HERE %s\n", v)
		// fmt.Printf("HERE %#v", v)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	// using the function
	fmt.Println(len(os.Args), os.Args)
	if slices.Contains(os.Args, "--dev") {
		curren, err := os.Getwd()
		check(err)
		currentDir = curren
	} else {
		exePath, err := os.Executable()
		currentDir = filepath.Dir(exePath)
		check(err)
	}
	configFilePAth := filepath.Join(currentDir, "configs.json")
	if _, err := os.Stat(configFilePAth); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("Config Not Exist on Path %s", configFilePAth))
	}
	dat, err := os.ReadFile(configFilePAth)
	check(err)
	json.Unmarshal(dat, ThisConfig)

	if errs := validator.Validator.Validate(ThisConfig); len(errs) > 0 {
		panic(fmt.Errorf("Config Error %#v", errs))
	}
	if ThisConfig.UseTextMessage {
		if !ThisConfig.ReadMessageFromCsv && len(ThisConfig.Message) == 0 {
			panic("Please Pass Message in Config File If you want to send Text Mesasge Or Make useTextMessage to false")
		}
	}
	Whatsapp()
}

func Whatsapp() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal

				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		println("Connected")
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
