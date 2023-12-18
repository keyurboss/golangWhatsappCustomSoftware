package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/golangWhatsappCustomSoftware/validator"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"

	// "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type Config struct {
	UseTextMessage                                 bool   `json:"useTextMessage" validate:"boolean"`
	AppendMessageToMedia                           bool   `json:"appendMessageToMedia" validate:"boolean"`
	ReadMessageFromCsv                             bool   `json:"readMessageFromCsv" validate:"boolean"`
	Message                                        string `json:"message"`
	AddMinimumDelayInSecondsAfterSuccessfulMessage int    `json:"addMinimumDelayInSecondsAfterSuccessfulMessage" validate:"required"`
	BasePathForAssets                              string `json:"basePathForAssets"`
	InputFileName                                  string `json:"inputFileName"`
}

var currentDir = ""
var InputFilePath = ""
var OutPutFilePath = ""
var ThisConfig = new(Config)
var client *whatsmeow.Client
var NonNumber, _ = regexp.Compile(`/\D/g`)

var LoopStarted = false

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	case *events.Connected:
		println("Client Connected")
		go AfterSuccessFullConnection()
	default:
		fmt.Printf("Event Occurred%s\n", reflect.TypeOf(v))
	}
}

func AppendToOutPutFile(text string) {
	f, err := os.OpenFile(OutPutFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
}

func AfterSuccessFullConnection() {
	if LoopStarted {
		println("Tried to Start Loop Again")
		return
	}
	LoopStarted = true
	time.Sleep(3 * time.Second)
	fmt.Printf("Reading File %s\n", InputFilePath)
	inputBytes, err := os.ReadFile(InputFilePath)
	check(err)
	input := string(inputBytes)
	RowsData := strings.Split(input, "\n")
	fmt.Printf("total %d Rows Found\n", len(RowsData))
	for _, row := range RowsData {
		cols := strings.Split(row, ",")
		if len(cols) < 2 {
			AppendToOutPutFile("Cells Length < 2 Found\n")
			continue
		}
		number := string(NonNumber.ReplaceAll([]byte(cols[0]), []byte("")))
		if len(number) < 10 {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Length %d of Number is Less than 10\n", number, len(number)))
			continue
		}
		fileName := fmt.Sprintf("%s.pdf", strings.TrimSpace(cols[1]))
		sendFilePath := filepath.Join(ThisConfig.BasePathForAssets, fileName)
		if _, err := os.Stat(sendFilePath); errors.Is(err, os.ErrNotExist) {
			AppendToOutPutFile(fmt.Sprintf("%s,false,File Path Not Exists %s\n", number, sendFilePath))
			continue
		}
		IsOnWhatsappCheck, err := client.IsOnWhatsApp([]string{"+" + number})
		if err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Something Went Wrong %#v\n", number, err))
			continue
		}
		NumberOnWhatsapp := IsOnWhatsappCheck[0]
		if !NumberOnWhatsapp.IsIn {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Number %s Not On Whatsapp\n", number, number))
			continue
		}
		pdfBytes, err := os.ReadFile(sendFilePath)
		if err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Reading File %#v\n", number, err))
			continue
		}
		println("Uploading File")
		resp, err := client.Upload(context.Background(), pdfBytes, whatsmeow.MediaDocument)
		if err != nil {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
			continue
		}
		docProto := &waProto.DocumentMessage{
			Url:           &resp.URL,
			Mimetype:      proto.String("application/pdf"),
			FileName:      &fileName,
			DirectPath:    &resp.DirectPath,
			MediaKey:      resp.MediaKey,
			FileEncSha256: resp.FileEncSHA256,
			FileSha256:    resp.FileSHA256,
			FileLength:    &resp.FileLength,
		}

		if ThisConfig.AppendMessageToMedia {
			if !ThisConfig.ReadMessageFromCsv {
				docProto.Caption = &ThisConfig.Message
			} else if ThisConfig.ReadMessageFromCsv && len(cols) >= 3 && len(cols[2]) > 0 {
				docProto.Caption = &cols[2]
			}
		}
		// targetJID := types.NewJID("917016879936", types.DefaultUserServer)
		targetJID := NumberOnWhatsapp.JID
		fmt.Printf("sending File To %s\n", number)
		client.SendMessage(context.TODO(), targetJID, &waProto.Message{
			DocumentMessage: docProto,
		})
		if !ThisConfig.AppendMessageToMedia {
			message := new(string)
			if !ThisConfig.ReadMessageFromCsv {
				message = &ThisConfig.Message
			} else if ThisConfig.ReadMessageFromCsv && len(cols) >= 3 && len(cols[2]) > 0 {
				message = &cols[2]
			}
			if len(*message) > 0 {
				fmt.Printf("sending Message To %s\n", number)
				client.SendMessage(context.TODO(), targetJID, &waProto.Message{
					Conversation: proto.String(*message),
				})
			}
		}
		AppendToOutPutFile(fmt.Sprintf("%s,true\n", number))
		time.Sleep(time.Second * time.Duration(ThisConfig.AddMinimumDelayInSecondsAfterSuccessfulMessage))
	}

	println("It is Completed")
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
		current, err := os.Getwd()
		check(err)
		currentDir = current
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
			panic("Please Pass Message in Config File If you want to send Text Message Or Make useTextMessage to false")
		}
	}
	if ThisConfig.BasePathForAssets == "" {
		ThisConfig.BasePathForAssets = filepath.Join(currentDir, "assets")
	}

	if _, err := os.Stat(ThisConfig.BasePathForAssets); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("base path for assets not exists %s", configFilePAth))
	}
	if len(ThisConfig.InputFileName) > 0 {
		InputFilePath = filepath.Join(currentDir, ThisConfig.InputFileName)
	} else {
		InputFilePath = filepath.Join(currentDir, "input.csv")
	}
	OutPutFilePath = filepath.Join(filepath.Dir(InputFilePath), "output.csv")
	if _, err := os.Stat(InputFilePath); errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("input File Not Exists at %s", InputFilePath))
	}
	Whatsapp()
}

func Whatsapp() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:WhatsappSuperSecrete.db?_foreign_keys=on", dbLog)
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
